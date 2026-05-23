package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dwlhm/nova/internal/dev"
)

func runDev(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	flags.SetOutput(stderr)
	targetID := flags.String("target", "web", "target to develop: web or android")
	outRoot := flags.String("out", ".", "output root")
	bundleTarget := flags.Bool("bundle", true, "run target bundling after artifact generation")
	offline := flags.Bool("offline", false, "run target bundling without dependency downloads when supported")
	gradlePath := flags.String("gradle", "", "gradle executable path for android bundling")
	gradleTask := flags.String("gradle-task", "assembleDebug", "gradle task for android bundling")
	adbPath := flags.String("adb", "", "adb executable path for android install/update")
	androidUser := flags.String("android-user", "0", "android user id for install/update and launch")
	addr := flags.String("addr", "127.0.0.1:0", "web dev server address")
	interval := flags.Duration("interval", time.Second, "file polling interval")
	launchAndroid := flags.Bool("launch", true, "launch android app after install/update")
	once := flags.Bool("once", false, "run one dev cycle and exit")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	target := dev.Target(*targetID)
	if target != dev.TargetWeb && target != dev.TargetAndroid {
		fmt.Fprintf(stderr, "NVA-TARGET-019: unsupported dev target %s\n", *targetID)
		return 1
	}
	if *interval <= 0 {
		fmt.Fprintln(stderr, "NVA-DEV-001: --interval must be greater than zero")
		return 1
	}
	if target == dev.TargetAndroid && strings.TrimSpace(*androidUser) == "" {
		fmt.Fprintln(stderr, "NVA-DEV-001: --android-user must not be empty")
		return 1
	}

	ctx := context.Background()
	var server *webDevServer
	if target == dev.TargetWeb && !*once {
		root := filepath.Join(cwd, filepath.FromSlash(*outRoot), "build", "web")
		started, err := startWebDevServer(root, *addr)
		if err != nil {
			fmt.Fprintf(stderr, "NVA-DEV-002: start web dev server: %s\n", err.Error())
			return 1
		}
		server = started
		defer server.close(ctx)
		fmt.Fprintf(stdout, "serving web dev mode at %s\n", server.url)
	}

	initial := dev.PlanCycle(dev.CycleInput{
		Target:  target,
		Initial: true,
		Launch:  *launchAndroid,
	})
	result, ok := runDevBuild(ctx, cwd, buildOptions{
		TargetID:     string(target),
		OutRoot:      *outRoot,
		BundleTarget: *bundleTarget && initial.Bundle,
		Offline:      *offline,
		GradlePath:   *gradlePath,
		GradleTask:   *gradleTask,
	}, initial, stdout, stderr)
	if ok && target == dev.TargetAndroid {
		ok = deployAndroid(ctx, cwd, result, *adbPath, *androidUser, initial.LaunchAndroid, stdout, stderr)
	}
	if *once {
		if ok {
			return 0
		}
		return 1
	}

	snapshot, err := snapshotProject(cwd)
	if err != nil {
		fmt.Fprintf(stderr, "NVA-DEV-001: snapshot project: %s\n", err.Error())
		return 1
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for range ticker.C {
		next, err := snapshotProject(cwd)
		if err != nil {
			fmt.Fprintf(stderr, "NVA-DEV-001: snapshot project: %s\n", err.Error())
			continue
		}
		changes := dev.DiffSnapshots(snapshot, next)
		if changes.Empty() {
			continue
		}
		snapshot = next
		fmt.Fprintf(stdout, "detected changes: %s\n", strings.Join(changes.Paths, ", "))

		plan := dev.PlanCycle(dev.CycleInput{
			Target:  target,
			Changes: changes,
			Launch:  *launchAndroid,
		})
		result, ok := runDevBuild(ctx, cwd, buildOptions{
			TargetID:     string(target),
			OutRoot:      *outRoot,
			BundleTarget: *bundleTarget && plan.Bundle,
			Offline:      *offline,
			GradlePath:   *gradlePath,
			GradleTask:   *gradleTask,
		}, plan, stdout, stderr)
		if !ok {
			continue
		}
		if target == dev.TargetWeb && server != nil && plan.ReloadBrowser {
			server.reload()
		}
		if target == dev.TargetAndroid {
			deployAndroid(ctx, cwd, result, *adbPath, *androidUser, plan.LaunchAndroid, stdout, stderr)
		}
	}
	return 0
}

func runDevBuild(ctx context.Context, cwd string, options buildOptions, plan dev.CyclePlan, stdout io.Writer, stderr io.Writer) (buildResult, bool) {
	if !plan.Build {
		return buildResult{}, true
	}
	fmt.Fprintf(stdout, "dev strategy: %s\n", plan.Strategy)
	return buildProject(ctx, cwd, options, stdout, stderr)
}

func deployAndroid(ctx context.Context, cwd string, result buildResult, adbPath string, androidUser string, launch bool, stdout io.Writer, stderr io.Writer) bool {
	if result.Bundle == nil {
		fmt.Fprintln(stderr, "NVA-DEV-003: android dev mode requires --bundle=true to install the debug APK")
		return false
	}
	applicationID := strings.TrimSpace(result.Project.Targets["android"].Options["application_id"])
	if applicationID == "" {
		fmt.Fprintln(stderr, "NVA-DEV-003: targets.android.application_id is required to launch android dev mode")
		return false
	}
	activityClass := strings.TrimSpace(result.Project.Targets["android"].Options["namespace"])
	if activityClass == "" {
		fmt.Fprintln(stderr, "NVA-DEV-003: targets.android.namespace is required to launch android dev mode")
		return false
	}
	activityClass += ".MainActivity"
	adb, err := selectADB(adbPath)
	if err != nil {
		fmt.Fprintf(stderr, "NVA-DEV-003: %s\n", err.Error())
		return false
	}
	userID := strings.TrimSpace(androidUser)
	install := dev.AndroidInstallCommand(adb, result.Bundle.OutputPath, userID)
	if err := runDevCommand(ctx, cwd, install, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "NVA-DEV-003: adb install failed: %s\n", err.Error())
		return false
	}
	fmt.Fprintf(stdout, "installed android app for user %s with %s\n", userID, strings.Join(install.Args, " "))
	if !launch {
		return true
	}
	start := dev.AndroidLaunchCommand(adb, applicationID, activityClass, userID)
	if err := runDevCommand(ctx, cwd, start, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "NVA-DEV-003: adb launch failed: %s\n", err.Error())
		return false
	}
	fmt.Fprintf(stdout, "launched android activity %s/%s for user %s\n", applicationID, activityClass, userID)
	return true
}

func runDevCommand(ctx context.Context, cwd string, spec dev.CommandSpec, stdout io.Writer, stderr io.Writer) error {
	if strings.TrimSpace(spec.Path) == "" {
		return errors.New("command path is empty")
	}
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	cmd.Dir = cwd
	var combined bytes.Buffer
	cmd.Stdout = capturedWriter(stdout, &combined)
	cmd.Stderr = capturedWriter(stderr, &combined)
	if err := cmd.Run(); err != nil {
		return err
	}
	if message, ok := commandOutputError(combined.String()); ok {
		return errors.New(message)
	}
	return nil
}

func capturedWriter(writer io.Writer, capture *bytes.Buffer) io.Writer {
	if writer == nil {
		return capture
	}
	return io.MultiWriter(writer, capture)
}

func commandOutputError(output string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Error:") {
			return line, true
		}
	}
	return "", false
}

func selectADB(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if value := os.Getenv("NOVA_ADB"); value != "" {
		return value, nil
	}
	for _, envName := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if root := os.Getenv(envName); root != "" {
			candidate := filepath.Join(root, "platform-tools", "adb")
			if isExecutableFile(candidate) {
				return candidate, nil
			}
		}
	}
	if path, err := exec.LookPath("adb"); err == nil {
		return path, nil
	}
	return "", errors.New("adb executable not found; set NOVA_ADB or pass --adb")
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func snapshotProject(cwd string) (dev.Snapshot, error) {
	snapshot := make(dev.Snapshot)
	err := filepath.WalkDir(cwd, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(cwd, path)
		if err != nil {
			return err
		}
		normalized := filepath.ToSlash(rel)
		if !isWatchedDevFile(normalized) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		snapshot[normalized] = dev.FileState{
			Path:    normalized,
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
		}
		return nil
	})
	return snapshot, err
}

func isWatchedDevFile(path string) bool {
	if path == "nova.toml" {
		return true
	}
	switch filepath.Ext(path) {
	case ".nova", ".css", ".js", ".ts", ".java", ".kts", ".toml":
		return true
	default:
		return false
	}
}

type webDevServer struct {
	url    string
	server *http.Server
	events *devEventBroker
}

func startWebDevServer(root string, addr string) (*webDevServer, error) {
	events := newDevEventBroker()
	mux := http.NewServeMux()
	server := &webDevServer{events: events}
	mux.HandleFunc("/__nova/dev/events", events.serveHTTP)
	mux.HandleFunc("/", serveWebDevFile(root))

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server.url = "http://" + listener.Addr().String()
	server.server = &http.Server{Handler: mux}
	go func() {
		if err := server.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "NVA-DEV-002: web dev server stopped: %s\n", err.Error())
		}
	}()
	return server, nil
}

func (server *webDevServer) reload() {
	server.events.broadcast("reload")
}

func (server *webDevServer) close(ctx context.Context) {
	_ = server.server.Shutdown(ctx)
}

func serveWebDevFile(root string) http.HandlerFunc {
	fileServer := http.FileServer(http.Dir(root))
	return func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" || request.URL.Path == "/index.html" {
			content, err := os.ReadFile(filepath.Join(root, "index.html"))
			if err != nil {
				http.NotFound(response, request)
				return
			}
			response.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(response, injectWebDevClient(string(content)))
			return
		}
		fileServer.ServeHTTP(response, request)
	}
}

func injectWebDevClient(index string) string {
	client := `<script>
(() => {
  const events = new EventSource("/__nova/dev/events");
  events.addEventListener("reload", () => window.location.reload());
})();
</script>
`
	if strings.Contains(index, "</body>") {
		return strings.Replace(index, "</body>", client+"</body>", 1)
	}
	return index + client
}

type devEventBroker struct {
	mu      sync.Mutex
	clients map[chan string]bool
}

func newDevEventBroker() *devEventBroker {
	return &devEventBroker{clients: make(map[chan string]bool)}
}

func (broker *devEventBroker) broadcast(event string) {
	broker.mu.Lock()
	clients := make([]chan string, 0, len(broker.clients))
	for client := range broker.clients {
		clients = append(clients, client)
	}
	broker.mu.Unlock()
	for _, client := range clients {
		select {
		case client <- event:
		default:
		}
	}
}

func (broker *devEventBroker) serveHTTP(response http.ResponseWriter, request *http.Request) {
	flusher, ok := response.(http.Flusher)
	if !ok {
		http.Error(response, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Connection", "keep-alive")

	client := make(chan string, 8)
	broker.add(client)
	defer broker.remove(client)

	_, _ = io.WriteString(response, ": connected\n\n")
	flusher.Flush()
	for {
		select {
		case <-request.Context().Done():
			return
		case event := <-client:
			_, _ = fmt.Fprintf(response, "event: %s\ndata: {}\n\n", event)
			flusher.Flush()
		}
	}
}

func (broker *devEventBroker) add(client chan string) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	broker.clients[client] = true
}

func (broker *devEventBroker) remove(client chan string) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	delete(broker.clients, client)
}

package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/packageio"
)

func runLock(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lock", flag.ContinueOnError)
	flags.SetOutput(stderr)
	targetID := flags.String("target", "web", "target used to resolve renderer adapters: web or android")
	check := flags.Bool("check", false, "verify nova.lock is up to date without writing")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	manifest, diagnostics, ok := loadManifest(cwd)
	for _, item := range diagnostics {
		fmt.Fprintln(stderr, item)
	}
	if !ok {
		return 1
	}

	lockfile, lockDiagnostics := packageio.GenerateLockfile(cwd, *targetID, manifest)
	for _, item := range lockDiagnostics {
		fmt.Fprintf(stderr, "%s: %s\n", item.Code, item.Message)
	}
	if diagnostic.HasErrors(lockDiagnostics) {
		return 1
	}

	if *check {
		existing, existingDiagnostics := packageio.LoadLockfile(cwd, false, manifest)
		for _, item := range existingDiagnostics {
			fmt.Fprintf(stderr, "%s: %s\n", item.Code, item.Message)
		}
		if diagnostic.HasErrors(existingDiagnostics) {
			return 1
		}
		if !packageio.LockfilesEqual(existing, lockfile) {
			fmt.Fprintln(stderr, "NVA-PKG-017: nova.lock is out of date; run nova lock")
			return 1
		}
		fmt.Fprintln(stdout, "nova.lock is up to date")
		return 0
	}

	if err := packageio.WriteLockfile(cwd, lockfile); err != nil {
		fmt.Fprintf(stderr, "NVA-PKG-016: write nova.lock: %s\n", err.Error())
		return 1
	}
	fmt.Fprintf(stdout, "wrote nova.lock with %d package(s)\n", len(lockfile.Entries))
	return 0
}

// Package contract defines the stable runtime application contract (bundle payload).
package contract

// Version is the application contract schema version embedded in App.V.
const Version = 1

// App is the only JSON shape loaded by production runtimes (web JS, tooling).
type App struct {
	V                  int                 `json:"v"`
	Target             string              `json:"target"`
	Entry              string              `json:"entry"`
	Permissions        []string            `json:"permissions,omitempty"`
	Model              Model               `json:"model"`
	View               View                `json:"view"`
	Effects            []Effect            `json:"effects,omitempty"`
	Events             []EventContract     `json:"events,omitempty"`
	Lifecycles         []Lifecycle         `json:"lifecycles,omitempty"`
	ExternalOperations []ExternalOperation `json:"externalOperations,omitempty"`
	Persistence        *HydrationManifest  `json:"persistence,omitempty"`
	AppLifecycle       []string            `json:"appLifecycle,omitempty"`
}

// HydrationManifest describes storage restore steps lowered from lifecycle metadata.
type HydrationManifest struct {
	BootstrapEvent  string              `json:"bootstrapEvent,omitempty"`
	Loads           []HydrationLoadStep `json:"loads,omitempty"`
	TerminalEvent   string              `json:"terminalEvent,omitempty"`
	SkipAfterEvents []string            `json:"skipAfterEvents,omitempty"`
}

// HydrationLoadStep is one storage load executed during mount restore.
type HydrationLoadStep struct {
	TriggerEvent string            `json:"triggerEvent"`
	EffectID     string            `json:"effectId"`
	Input        map[string]string `json:"input,omitempty"`
	SuccessEvent string            `json:"successEvent"`
	FailureEvent string            `json:"failureEvent,omitempty"`
}

type Model struct {
	States []State `json:"states"`
}

type State struct {
	Owner       string       `json:"owner"`
	Name        string       `json:"name"`
	Type        string       `json:"type,omitempty"`
	Initial     string       `json:"initial"`
	Transitions []Transition `json:"transitions"`
}

type Transition struct {
	Event      string   `json:"event"`
	Params     []string `json:"params,omitempty"`
	Expression string   `json:"expression"`
}

type View struct {
	Nodes    []Node        `json:"nodes"`
	Bindings []BindingMeta `json:"bindings,omitempty"`
}

type BindingMeta struct {
	At     []int    `json:"at"`
	Prop   string   `json:"prop"`
	States []string `json:"states"`
}

type Node struct {
	Kind     string            `json:"kind"`
	Props    map[string]string `json:"props,omitempty"`
	Events   map[string]Event  `json:"events,omitempty"`
	Children []Node            `json:"children,omitempty"`
	Key      string            `json:"key,omitempty"`
}

type Event struct {
	Name string   `json:"name"`
	Args []string `json:"args,omitempty"`
}

// Effect is a resolved external port referenced from lifecycle (audit + runtime wiring).
type Effect struct {
	ID          string   `json:"id"`
	Permissions []string `json:"permissions,omitempty"`
}

// EventContract describes a scheduler event allowed at runtime.
type EventContract struct {
	Name     string   `json:"name"`
	Emitters []string `json:"emitters,omitempty"`
	Payload  string   `json:"payload,omitempty"`
}

// Lifecycle is an executable lifecycle handler lowered from <lifecycle> blocks.
type Lifecycle struct {
	Owner string          `json:"owner"`
	Phase string          `json:"phase"`
	Event string          `json:"event,omitempty"`
	Steps []LifecycleStep `json:"steps"`
}

// LifecycleStep is one emit or external invocation inside a lifecycle handler.
type LifecycleStep struct {
	Emit     *LifecycleEmit     `json:"emit,omitempty"`
	External *LifecycleExternal `json:"external,omitempty"`
}

// LifecycleEmit schedules an event from lifecycle output.
type LifecycleEmit struct {
	Name string   `json:"name"`
	Args []string `json:"args,omitempty"`
}

// LifecycleExternal invokes a resolved external port from lifecycle output.
type LifecycleExternal struct {
	EffectID  string            `json:"effectId"`
	Input     map[string]string `json:"input,omitempty"`
	OnSuccess string            `json:"onSuccess,omitempty"`
	OnFailure string            `json:"onFailure,omitempty"`
}

// ExternalOperation is the adapter contract for one resolved external port.
type ExternalOperation struct {
	ID             string   `json:"id"`
	Capability     string   `json:"capability,omitempty"`
	Source         string   `json:"source"`
	Operation      string   `json:"operation"`
	Output         string   `json:"output,omitempty"`
	Permissions    []string `json:"permissions,omitempty"`
	Implementation string   `json:"implementation,omitempty"`
}

// BuildManifest is dev/CI metadata; not required at runtime.
type BuildManifest struct {
	ContractVersion    int      `json:"contractVersion"`
	LanguageVersion    string   `json:"languageVersion"`
	SchedulerVersion   string   `json:"schedulerVersion"`
	RuntimeVersion     string   `json:"runtimeVersion"`
	Target             string   `json:"target"`
	Entry              string   `json:"entry"`
	Modules            []string `json:"modules,omitempty"`
	TemplateFile       string   `json:"templateFile,omitempty"`
	TemplateIndex      int      `json:"templateIndex,omitempty"`
	ExternalOperations []string `json:"externalOperations,omitempty"`
	Permissions        []string `json:"permissions,omitempty"`
}

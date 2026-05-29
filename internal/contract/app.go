// Package contract defines the stable runtime application contract (bundle payload).
package contract

// Version is the application contract schema version embedded in App.V.
const Version = 1

// App is the only JSON shape loaded by production runtimes (web JS, tooling).
type App struct {
	V           int      `json:"v"`
	Target      string   `json:"target"`
	Entry       string   `json:"entry"`
	Permissions []string `json:"permissions,omitempty"`
	Model       Model    `json:"model"`
	View        View     `json:"view"`
	Effects     []Effect `json:"effects,omitempty"`
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

package project

type Project struct {
	Name    string
	Version string
	Entry   string
}

type Target struct {
	Renderer     string
	Options      map[string]string
	Styles       []string
	ScopedStyles []string
}

type RendererUnknownKindPolicy string

const (
	RendererUnknownKindError          RendererUnknownKindPolicy = "error"
	RendererUnknownKindWarn           RendererUnknownKindPolicy = "warn"
	RendererUnknownKindPassthroughWeb RendererUnknownKindPolicy = "passthrough_web"
)

type RendererConfig struct {
	UnknownKind       RendererUnknownKindPolicy
	ExtensionPackages []RendererPackageRef
	Dictionary        []RendererPrimitive
}

type RendererPackageRef struct {
	Name       string
	Constraint string
}

type RendererPrimitive struct {
	Kind          string
	Description   string
	Props         []RendererField
	Events        []RendererEvent
	AllowOverride bool
	Targets       map[string]RendererTarget
}

type RendererField struct {
	Name        string
	Type        string
	Optional    bool
	Description string
}

type RendererEvent struct {
	Name        string
	Payload     string
	Description string
}

type RendererTarget struct {
	Strategy string
	Adapter  string
	Tag      string
	Delegate string
}

type PermissionMap map[string]bool

type Dependency struct {
	Name       string
	Constraint string
}

type Manifest struct {
	Project          Project
	Dependencies     []Dependency
	Targets          map[string]Target
	Renderer         RendererConfig
	Permissions      PermissionMap
	PermissionScopes map[string][]string
}

type File struct {
	Path string
}

type Diagnostic struct {
	Code    string
	Message string
	Path    string
}

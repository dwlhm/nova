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

type PermissionMap map[string]bool

type Manifest struct {
	Project          Project
	Targets          map[string]Target
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

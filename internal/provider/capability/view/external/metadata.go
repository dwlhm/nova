package external

// Role tells the host how to orchestrate a capability module.
type Role string

const (
	RolePrimitive    Role = "primitive"
	RoleShell        Role = "shell"
	RoleCrossCutting Role = "cross_cutting"
)

// Metadata describes when and how internal host should invoke a capability.
type Metadata struct {
	ID       string
	Role     Role
	Kinds    []string
	Targets  []string
	Order    int
	Requires []string
}

func (meta Metadata) SupportsTarget(target string) bool {
	for _, entry := range meta.Targets {
		if entry == target {
			return true
		}
	}
	return false
}

func (meta Metadata) HandlesKind(kind string) bool {
	for _, entry := range meta.Kinds {
		if entry == kind {
			return true
		}
	}
	return false
}

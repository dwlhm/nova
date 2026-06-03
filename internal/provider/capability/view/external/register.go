package external

func PrimitiveMetadata(id string, kinds []string, order int) Metadata {
	return Metadata{
		ID:      id,
		Role:    RolePrimitive,
		Kinds:   kinds,
		Targets: []string{"android", "web"},
		Order:   order,
	}
}

func ShellMetadata(id string, order int, targets ...string) Metadata {
	if len(targets) == 0 {
		targets = []string{"android", "web"}
	}
	return Metadata{
		ID:      id,
		Role:    RoleShell,
		Targets: targets,
		Order:   order,
	}
}

func CrossCuttingMetadata(id string, order int, targets ...string) Metadata {
	if len(targets) == 0 {
		targets = []string{"android", "web"}
	}
	return Metadata{
		ID:      id,
		Role:    RoleCrossCutting,
		Targets: targets,
		Order:   order,
	}
}

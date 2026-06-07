package types

// MergeEnvironments returns an environment with definitions from base and extra.
// Definitions in base take precedence over extra when names collide.
func MergeEnvironments(base Environment, extra Environment) Environment {
	merged := Environment{
		Definitions:                   make(map[string]Type, len(base.Definitions)+len(extra.Definitions)),
		AllowUnknownNamedSerializable: base.AllowUnknownNamedSerializable || extra.AllowUnknownNamedSerializable,
	}
	for name, typ := range extra.Definitions {
		merged.Definitions[name] = typ
	}
	for name, typ := range base.Definitions {
		merged.Definitions[name] = typ
	}
	return merged
}

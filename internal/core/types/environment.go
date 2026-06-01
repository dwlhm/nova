package types

import "github.com/dwlhm/nova/internal/core/parser"

func EmptyEnvironment() Environment {
	return Environment{
		Definitions:                   make(map[string]Type),
		AllowUnknownNamedSerializable: true,
	}
}

func BuildEnvironment(file parser.File) (Environment, []Diagnostic) {
	env := EmptyEnvironment()
	diagnostics := make([]Diagnostic, 0)

	for _, decl := range file.ContractTypes {
		if decl.Name == "" {
			continue
		}
		env.Definitions[decl.Name] = Type{Kind: KindNamed, Name: decl.Name}
	}

	for _, decl := range file.ContractTypes {
		if decl.Name == "" {
			continue
		}
		switch {
		case decl.Opaque:
			env.Definitions[decl.Name] = Type{Kind: KindOpaque, Name: decl.Name}
		case len(decl.Fields) > 0:
			fields := make([]Field, 0, len(decl.Fields))
			for _, field := range decl.Fields {
				fieldType, fieldDiagnostics := ParseRef(env, field.Type)
				diagnostics = append(diagnostics, fieldDiagnostics...)
				fields = append(fields, Field{
					Name:     field.Name,
					Optional: field.Optional,
					Type:     fieldType,
				})
			}
			env.Definitions[decl.Name] = Type{Kind: KindRecord, Name: decl.Name, Fields: fields}
		default:
			alias, aliasDiagnostics := ParseRef(env, decl.Alias)
			diagnostics = append(diagnostics, aliasDiagnostics...)
			env.Definitions[decl.Name] = alias
		}
	}

	return env, diagnostics
}

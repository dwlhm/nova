package packages

import "sort"

func rendererPrimitives(manifest Manifest) []RendererPrimitive {
	kinds := make([]string, 0, len(manifest.Renderer.Primitives))
	for kind := range manifest.Renderer.Primitives {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)

	primitives := make([]RendererPrimitive, 0, len(kinds))
	for _, kind := range kinds {
		primitive := manifest.Renderer.Primitives[kind]
		if primitive.Kind == "" {
			primitive.Kind = kind
		}
		if primitive.Package == "" {
			primitive.Package = manifest.Name
		}
		primitives = append(primitives, primitive)
	}
	return primitives
}

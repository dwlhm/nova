package lsp

import (
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/provider/standard"
)

const (
	lspCompletionProperty = 10
	lspCompletionClass    = 7
)

type builtinIndex struct {
	byName map[string]standard.RendererPrimitive
}

func builtins() builtinIndex {
	return builtinIndexFromPrimitives(standard.RendererPrimitives())
}

func builtinsFromPackageGraph(graph packages.ResolvedGraph) builtinIndex {
	return builtinIndexFromPrimitives(standard.RendererPrimitivesFromPackages(graph))
}

func builtinIndexFromPrimitives(primitives []standard.RendererPrimitive) builtinIndex {
	byName := make(map[string]standard.RendererPrimitive, len(primitives))
	for _, primitive := range primitives {
		byName[primitive.Name] = primitive
	}
	return builtinIndex{byName: byName}
}

func (index builtinIndex) primitive(name string) (standard.RendererPrimitive, bool) {
	primitive, ok := index.byName[name]
	return primitive, ok
}

func (index builtinIndex) attribute(nodeName string, attributeName string) (standard.RendererPrimitive, standard.PrimitiveField, standard.PrimitiveEvent, bool) {
	primitive, ok := index.primitive(nodeName)
	if !ok {
		return standard.RendererPrimitive{}, standard.PrimitiveField{}, standard.PrimitiveEvent{}, false
	}
	for _, prop := range primitive.Props {
		if prop.Name == attributeName {
			return primitive, prop, standard.PrimitiveEvent{}, true
		}
	}
	for _, event := range primitive.Events {
		if event.Name == attributeName {
			return primitive, standard.PrimitiveField{}, event, true
		}
	}
	return standard.RendererPrimitive{}, standard.PrimitiveField{}, standard.PrimitiveEvent{}, false
}

func (index builtinIndex) nodeCompletions() []CompletionItem {
	names := make([]string, 0, len(index.byName))
	for name := range index.byName {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]CompletionItem, 0, len(names))
	for _, name := range names {
		primitive := index.byName[name]
		items = append(items, CompletionItem{
			Label:         primitive.Name,
			Kind:          lspCompletionClass,
			Detail:        primitive.Package + " primitive",
			Documentation: markup(primitiveMarkdown(primitive)),
			InsertText:    primitive.Name,
			SortText:      "20_" + primitive.Name,
		})
	}
	return items
}

func (index builtinIndex) attributeCompletions(nodeName string) []CompletionItem {
	primitive, ok := index.primitive(nodeName)
	if !ok {
		return commonAttributeCompletions()
	}
	items := make([]CompletionItem, 0, len(primitive.Props)+len(primitive.Events))
	for _, prop := range primitive.Props {
		insert := prop.Name + " <- "
		items = append(items, CompletionItem{
			Label:         insert,
			Kind:          lspCompletionProperty,
			Detail:        fieldDetail(prop),
			Documentation: markup(prop.Description),
			InsertText:    insert,
			SortText:      "10_" + prop.Name,
		})
	}
	for _, event := range primitive.Events {
		insert := event.Name + " -> "
		items = append(items, CompletionItem{
			Label:         insert,
			Kind:          lspCompletionEvent,
			Detail:        eventDetail(event),
			Documentation: markup(event.Description),
			InsertText:    insert,
			SortText:      "09_" + event.Name,
		})
	}
	return items
}

func commonAttributeCompletions() []CompletionItem {
	return []CompletionItem{
		{
			Label:      "class <- ",
			Kind:       lspCompletionProperty,
			Detail:     "class?: string",
			InsertText: "class <- ",
			SortText:   "10_class",
		},
		{
			Label:      "key <- ",
			Kind:       lspCompletionProperty,
			Detail:     "key?: unknown",
			InsertText: "key <- ",
			SortText:   "10_key",
		},
	}
}

func primitiveHover(primitive standard.RendererPrimitive) string {
	return "```nova\n" + primitive.Name + "  // " + primitive.Package + "\n```\n" + primitiveMarkdown(primitive)
}

func primitiveMarkdown(primitive standard.RendererPrimitive) string {
	parts := []string{primitive.Description}
	if len(primitive.Props) > 0 {
		props := make([]string, 0, len(primitive.Props))
		for _, prop := range primitive.Props {
			props = append(props, "`"+fieldDetail(prop)+"`")
		}
		parts = append(parts, "Props: "+strings.Join(props, ", "))
	}
	if len(primitive.Events) > 0 {
		events := make([]string, 0, len(primitive.Events))
		for _, event := range primitive.Events {
			events = append(events, "`"+eventDetail(event)+"`")
		}
		parts = append(parts, "Events: "+strings.Join(events, ", "))
	}
	return strings.Join(nonEmpty(parts), "\n\n")
}

func attributeHover(primitive standard.RendererPrimitive, prop standard.PrimitiveField, event standard.PrimitiveEvent) string {
	if prop.Name != "" {
		text := "```nova\n" + primitive.Name + "." + fieldDetail(prop) + "\n```"
		if prop.Description != "" {
			text += "\n" + prop.Description
		}
		return text
	}
	text := "```nova\n" + primitive.Name + "." + eventDetail(event) + "\n```"
	if event.Description != "" {
		text += "\n" + event.Description
	}
	return text
}

func fieldDetail(prop standard.PrimitiveField) string {
	optional := ""
	if prop.Optional {
		optional = "?"
	}
	return prop.Name + optional + ": " + prop.Type
}

func eventDetail(event standard.PrimitiveEvent) string {
	payload := event.Payload
	if payload == "" {
		payload = "void"
	}
	return event.Name + " -> " + payload
}

func markup(value string) MarkupContent {
	return MarkupContent{Kind: markupKindMarkdown, Value: value}
}

func nonEmpty(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

# nova-style-java

Android portable style applier embedded into generated apps as `NovaStyle.java`.

- `applyWithStates` — static class lists with pseudo-state color/background
- `applyDynamicClasses` — runtime `class` binding lookup via `NovaStyleRules.CLASS_RULES`
- `bindStatefulLayout` / `applyLayoutSkin` — layout props that change on `:active`/`:focus`/etc.

Semantic contract: [ADR-012](../../docs/adr/adr_012_style_format.md).

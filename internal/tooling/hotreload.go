package tooling

type PatchType string

const (
	PatchTemplate        PatchType = "template_patch"
	PatchPureFunc        PatchType = "pure_func_patch"
	PatchStyleToken      PatchType = "style_token_patch"
	PatchTargetAdapter   PatchType = "target_adapter_patch"
	PatchStateSchema     PatchType = "state_schema_patch"
	PatchPermission      PatchType = "permission_patch"
	PatchRuntimeSemantic PatchType = "runtime_semantic_patch"
)

type ReloadAction string

const (
	ReloadPatch       ReloadAction = "patch"
	ReloadFullReload  ReloadAction = "full_reload"
	ReloadFullRebuild ReloadAction = "full_rebuild"
)

type ABIDiff struct {
	PatchTypes               []PatchType
	HasExplicitMigration     bool
	EventContractChanged     bool
	FunctionSignatureChanged bool
}

type ReloadPlan struct {
	Action                  ReloadAction
	PreserveSnapshot        bool
	RequiresPermissionAudit bool
	Reasons                 []string
}

func PlanHotReload(diff ABIDiff) ReloadPlan {
	plan := ReloadPlan{Action: ReloadPatch, PreserveSnapshot: true}
	for _, patch := range diff.PatchTypes {
		plan = applyPatchRule(plan, patch, diff)
	}
	if diff.EventContractChanged || diff.FunctionSignatureChanged {
		plan = requireFullReload(plan, "ABI contract changed")
	}
	return plan
}

func applyPatchRule(plan ReloadPlan, patch PatchType, diff ABIDiff) ReloadPlan {
	switch patch {
	case PatchTemplate, PatchPureFunc, PatchStyleToken:
		return plan
	case PatchStateSchema:
		if diff.HasExplicitMigration {
			return requireFullReload(plan, "state schema changed with explicit migration")
		}
		return requireFullReloadWithoutSnapshot(plan, "state schema changed without migration")
	case PatchPermission:
		return requireFullRebuildWithAudit(plan, "permission graph changed")
	case PatchTargetAdapter:
		return requireFullRebuildWithAudit(plan, "target adapter changed")
	case PatchRuntimeSemantic:
		return requireFullReloadWithoutSnapshot(plan, "runtime semantic changed")
	default:
		return requireFullReload(plan, "unknown patch type")
	}
}

func requireFullReload(plan ReloadPlan, reason string) ReloadPlan {
	if plan.Action != ReloadFullRebuild {
		plan.Action = ReloadFullReload
	}
	plan.Reasons = append(plan.Reasons, reason)
	return plan
}

func requireFullReloadWithoutSnapshot(plan ReloadPlan, reason string) ReloadPlan {
	plan = requireFullReload(plan, reason)
	plan.PreserveSnapshot = false
	return plan
}

func requireFullRebuildWithAudit(plan ReloadPlan, reason string) ReloadPlan {
	plan.Action = ReloadFullRebuild
	plan.PreserveSnapshot = false
	plan.RequiresPermissionAudit = true
	plan.Reasons = append(plan.Reasons, reason)
	return plan
}

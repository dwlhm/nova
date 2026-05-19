package tooling

import "testing"

func TestPlanHotReloadPreservesStateForCompatibleTemplateAndFuncPatches(t *testing.T) {
	plan := PlanHotReload(ABIDiff{PatchTypes: []PatchType{PatchTemplate, PatchPureFunc}})

	if plan.Action != ReloadPatch {
		t.Fatalf("action = %s, want patch", plan.Action)
	}
	if !plan.PreserveSnapshot {
		t.Fatalf("compatible template and pure func patch should preserve snapshot")
	}
	if plan.RequiresPermissionAudit {
		t.Fatalf("compatible template and pure func patch should not require permission audit")
	}
}

func TestPlanHotReloadRequiresFullReloadOrAuditForAbiBreakingPatches(t *testing.T) {
	statePlan := PlanHotReload(ABIDiff{PatchTypes: []PatchType{PatchStateSchema}})
	if statePlan.Action != ReloadFullReload || statePlan.PreserveSnapshot {
		t.Fatalf("state schema patch plan = %+v, want full reload without snapshot preservation", statePlan)
	}

	permissionPlan := PlanHotReload(ABIDiff{PatchTypes: []PatchType{PatchPermission}})
	if permissionPlan.Action != ReloadFullRebuild || !permissionPlan.RequiresPermissionAudit {
		t.Fatalf("permission patch plan = %+v, want full rebuild with permission audit", permissionPlan)
	}

	adapterPlan := PlanHotReload(ABIDiff{PatchTypes: []PatchType{PatchTargetAdapter}})
	if adapterPlan.Action != ReloadFullRebuild || !adapterPlan.RequiresPermissionAudit {
		t.Fatalf("target adapter patch plan = %+v, want full rebuild with permission audit", adapterPlan)
	}
}

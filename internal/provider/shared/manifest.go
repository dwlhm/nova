package shared

import (
	"sort"

	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/ir"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/provider/build"
)

func BuildManifest(bundle ir.Bundle, plan build.BuildPlan, versions ManifestVersions) contract.BuildManifest {
	return contract.BuildManifest{
		ContractVersion:    contract.Version,
		LanguageVersion:    versions.LanguageVersion,
		SchedulerVersion:   versions.SchedulerVersion,
		RuntimeVersion:     versions.RuntimeVersion,
		Target:             bundle.App.Target,
		Entry:              bundle.App.Entry,
		Modules:            bundle.Modules,
		TemplateFile:       plan.Template.SourceFile,
		TemplateIndex:      plan.Template.Index,
		ExternalOperations: ExternalOperationNames(plan.ExternalOperations),
		Permissions:        ManifestPermissionStrings(plan.Permissions),
	}
}

func ManifestPermissionStrings(permissions []security.Permission) []string {
	if len(permissions) == 0 {
		return nil
	}
	out := make([]string, len(permissions))
	for i, permission := range permissions {
		out[i] = string(permission)
	}
	sort.Strings(out)
	return out
}

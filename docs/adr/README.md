# Architecture Decision Records

Nova memakai ADR untuk keputusan desain yang mengikat implementasi. Mulai transisi production,
baca [adr_000_production_baseline.md](adr_000_production_baseline.md) terlebih dahulu.

## Terminologi

| Istilah | Arti |
| --- | --- |
| **production** / **production v1** | Perilaku dan target yang didukung hari ini (`web`, `@nova/android`) |
| **v1** | Ruang lingkup fitur yang sengaja dibatasi; bukan "belum selesai" |
| **deprecated** | Masih ada di repo untuk migrasi (`@nova/android-compose`) |
| **Out of production v1 scope** | Sengaja ditunda; bukan keputusan final menolak fitur |

Semua ADR di bawah ini telah dirapikan dari istilah "MVP" ke terminologi di atas, kecuali
adr_000 yang merujuk fase eksperimen secara historis.

## Indeks ADR

| ADR | Topik | Status |
| --- | --- | --- |
| [000](adr_000_production_baseline.md) | Production baseline, layer trimming | Accepted |
| [001](adr_001_language_specification.md) | Bahasa Nova | Implemented / Accepted |
| [002](adr_002_scheduler_runtime_semantics.md) | Scheduler semantics | Implemented / Accepted |
| [003](adr_003_capability_module_system.md) | Capability & module graph | Implemented / Accepted |
| [004](adr_004_type_system.md) | Type system | Implemented / Accepted |
| [005](adr_005_effect_lifecycle_model.md) | Effect & lifecycle | Implemented / Accepted |
| [006](adr_006_template_view_model.md) | Template & ViewIR | Implemented / Accepted |
| [007](adr_007_platform_capability_model.md) | Platform capability | Implemented / Accepted |
| [008](adr_008_renderer_lowering_pipeline.md) | Renderer lowering | Implemented / Accepted |
| [009](adr_009_external_interop_model.md) | External interop & @env | Implemented / Accepted |
| [010](adr_010_build_target_resolution.md) | Build & target resolution | Implemented / Accepted |
| [011](adr_011_error_handling_diagnostics.md) | Diagnostics | Implemented / Accepted |
| [012](adr_012_project_layout_package_convention.md) | Project layout | Implemented / Accepted |
| [013](adr_013_security_permission_model.md) | Security & permissions | Implemented / Accepted |
| [014](adr_014_framework_runtime_architecture.md) | Framework architecture | Implemented / Accepted |
| [015](adr_015_standard_package_surface.md) | Standard packages `@nova/*` | Implemented / Accepted |
| [016](adr_016_web_target_runtime.md) | Web target | Implemented / Accepted |
| [017](adr_017_android_target_runtime.md) | Android target (Java default) | Implemented / Accepted |
| [018](adr_018_app_lifecycle_navigation_model.md) | Lifecycle & navigation | Implemented / Accepted |
| [019](adr_019_state_persistence_hydration_model.md) | Persistence & hydration | Implemented / Accepted |
| [020](adr_020_testing_conformance_strategy.md) | Testing & conformance | Implemented / Accepted |
| [021](adr_021_developer_tooling_hot_reload.md) | CLI, dev, hot reload | Implemented / Accepted |
| [022](adr_022_package_distribution_versioning.md) | Package distribution | Implemented / Accepted |
| [023](adr_023_page_projection_routing.md) | Page projection & routing | Implemented / Accepted |
| [024](adr_024_style_asset_injection.md) | Style asset injection | Implemented / Accepted |
| [025](adr_025_android_apk_size_renderer_strategy.md) | Android APK size & renderer | Accepted |

## Renderer defaults (production)

```toml
[targets.web]
renderer = "@nova/web"

[targets.android]
renderer = "@nova/android"   # Java native View, no Kotlin/Compose
```

Compatibility only:

```toml
renderer = "@nova/android-compose"   # deprecated Kotlin + Compose path
```

## Menambah ADR baru

1. Salin struktur ADR yang ada (Status, Context, Decision, Consequences).
2. Gunakan **Accepted** atau **Implemented / Accepted** bila sudah diimplementasi.
3. Hindari "MVP" sebagai label target akhir; gunakan **production v1** atau **Out of production v1 scope**.
4. Perubahan ABI/scheduler/ViewIR wajib update conformance fixture terkait.

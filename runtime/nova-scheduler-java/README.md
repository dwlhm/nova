# nova-scheduler-java

Production scheduler library for Nova Android targets. Implements [ADR-002](../../docs/adr/adr_002_scheduler_runtime_semantics.md) queue, envelope, and state commit semantics.

The generated `MainActivity` implements `NovaScheduler.Host` and delegates evaluation, route shaping, and view updates to app-specific code.

## Install

### Gradle (composite / generated Nova Android project)

```kotlin
// settings.gradle.kts
include(":nova-scheduler")

// app/build.gradle.kts
dependencies {
    implementation(project(":nova-scheduler"))
}
```

Nova artifact generation copies this library into `build/android/nova-scheduler/` and wires the dependency automatically.

### Standalone development

Requires Gradle 8+ (or generate a wrapper once with `gradle wrapper`):

```bash
gradle test
# or, after wrapper generation:
./gradlew test
```

Go embed tests in `embed_test.go` also verify scheduler sources ship with the Nova module.

## Package

All types live in `nova.scheduler`:

| Type | Role |
| --- | --- |
| `NovaScheduler` | Queue, drain, and state commit |
| `NovaScheduler.Host` | Host callbacks for evaluate/route/renderer |
| `NovaTransition` | Transition metadata from app IR |
| `NovaEventEnvelope` | Internal queued event (package-private) |

## Versioning

Library version tracks Nova `schedulerVersion` metadata (`0.1.0` in artifact `metadata.json`).

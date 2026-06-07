# Nova target runtime libraries

Production scheduler and renderer runtime implementations live here as installable libraries. The Nova compiler
embeds these sources through `runtime/*/bind` and copies them into generated artifacts; you can
also depend on them directly when building custom host runtimes.

| Library | Target | Install |
| --- | --- | --- |
| [nova-scheduler-js](nova-scheduler-js/) | Web (browser) | `npm install @nova/scheduler` |
| [nova-renderer-js](nova-renderer-js/) | Web (browser) | `npm install @nova/renderer` |
| [nova-scheduler-java](nova-scheduler-java/) | Android (JVM) | Gradle `implementation(project(":nova-scheduler"))` |
| [nova-runtime-java](nova-runtime-java/) | Android (JVM) | Copied into app namespace as `NovaRuntime.java` |

Semantic contract: [ADR-002](../docs/adr/adr_002_runtime.md).

Go reference scheduler (conformance only): `internal/scheduler`.

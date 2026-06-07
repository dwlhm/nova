# nova-runtime-java

Android expression evaluator and route helpers embedded into generated app artifacts.

The Nova compiler copies `NovaRuntime.java` into each app's generated namespace via
`runtimejava.Source(packageName)`. Behavioral changes belong here—not in Go string templates
under `internal/provider/.../codegen`.

Semantic contract: ADR-013 (lowered expressions evaluated at runtime).

Install (Gradle module integration is future work; today the source is copied per artifact):

```kotlin
// Generated artifact includes app-local NovaRuntime.java
```

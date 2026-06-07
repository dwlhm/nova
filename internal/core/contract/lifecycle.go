package contract

// StandardAppLifecycleEvents returns canonical @nova/app lifecycle events.
func StandardAppLifecycleEvents() []string {
	return []string{
		"@app_started",
		"@app_resumed",
		"@app_paused",
		"@app_stopped",
		"@app_restored",
	}
}

// StandardNavigationPlatformEmitters are runtime sources allowed to emit route events.
func StandardNavigationPlatformEmitters() []string {
	return []string{"platform", "renderer", "@nova/navigation"}
}

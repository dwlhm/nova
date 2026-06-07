package web

import appjs "github.com/dwlhm/nova/runtime/nova-app-js"

func appLifecycleModule() string {
	return appjs.LifecycleJS()
}

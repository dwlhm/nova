package androidcodegen

import "github.com/dwlhm/nova/internal/core/contract"

func javaSchedulerActivityHost() string {
	return `    private void dispatch(String eventName, List<Object> args) {
        scheduler.dispatch("renderer", eventName, args);
    }

    @Override
    public Map<String, Object> schedulerState() {
        return state;
    }

    @Override
    public List<NovaTransition> schedulerTransitions() {
        return transitions();
    }

    @Override
    public List<String> schedulerRoutePatterns() {
        return routePatterns();
    }

    @Override
    public List<String> schedulerStateNames() {
        return stateNames();
    }

    @Override
    public boolean schedulerHasRouteState() {
        return hasRouteState();
    }

    @Override
    public Object schedulerCloneRoute(Object value) {
        return cloneRoute(value);
    }

    @Override
    public Object schedulerRouteValueForShape(Object value, List<String> patterns) {
        return routeValueForShape(value, patterns);
    }

    @Override
    public Object schedulerEvaluate(String expression, Map<String, Object> state, Map<String, Object> payload) {
        if ("` + contract.NavigationApplyExpression + `".equals(expression)) {
            return NovaNavigation.applyRoute(routeBackStack, routePatterns(), state, payload);
        }
        return evaluate(expression, state, payload);
    }

    @Override
    public void schedulerReconcileRouteBackStack(Object beforeRoute, Object afterRoute) {
        reconcileRouteBackStack(beforeRoute, afterRoute);
    }

`
}

func javaSchedulerLifecycleStubs() string {
	return `    @Override
    public void schedulerBeforeEvent(String eventName, List<Object> args) {}

    @Override
    public void schedulerAfterEvent(String eventName, List<Object> args) {}

    protected void schedulerMountLifecycles() {}

    protected void schedulerDisposeLifecycles() {}

`
}

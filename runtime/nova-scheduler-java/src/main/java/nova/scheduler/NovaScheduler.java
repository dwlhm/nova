package nova.scheduler;

import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Nova production scheduler for Android/Java runtimes (ADR-002).
 */
public final class NovaScheduler {
    public interface Host {
        Map<String, Object> schedulerState();

        List<NovaTransition> schedulerTransitions();

        List<String> schedulerRoutePatterns();

        List<String> schedulerStateNames();

        boolean schedulerHasRouteState();

        Object schedulerCloneRoute(Object value);

        Object schedulerRouteValueForShape(Object value, List<String> patterns);

        Object schedulerEvaluate(String expression, Map<String, Object> state, Map<String, Object> payload);

        void schedulerReconcileRouteBackStack(Object beforeRoute, Object afterRoute);

        void schedulerApplyStateCommit(Set<String> invalidations);

        default void schedulerBeforeEvent(String eventName, List<Object> args) {}

        default void schedulerAfterEvent(String eventName, List<Object> args) {}
    }

    private final Host host;
    private final List<NovaEventEnvelope> queue = new ArrayList<>();
    private long nextSequence = 1L;
    private boolean draining = false;

    public NovaScheduler(Host host) {
        this.host = host;
    }

    public void dispatch(String source, String eventName, List<Object> args) {
        NovaEventEnvelope envelope = enqueue(source, eventName, args);
        if (envelope == null) {
            return;
        }
        drain();
    }

    public void enqueueLifecycle(String source, String eventName, List<Object> args) {
        NovaEventEnvelope envelope = enqueue(source, eventName, args);
        if (envelope != null && draining) {
            step();
        }
    }

    public void commitTransition(String eventName, List<Object> args) {
        if (eventName == null || !eventName.startsWith("@") || eventName.length() <= 1) {
            return;
        }
        List<Object> eventArgs = args == null ? Collections.emptyList() : new ArrayList<>(args);
        NovaEventEnvelope event = new NovaEventEnvelope(nextSequence++, "hydrate", eventName, eventArgs);
        Map<String, Object> beforeState = new LinkedHashMap<>(host.schedulerState());
        Object beforeRoute = host.schedulerCloneRoute(host.schedulerState().get("route"));
        Set<String> invalidations = applyTransitionCommit(event, beforeState);
        host.schedulerReconcileRouteBackStack(beforeRoute, host.schedulerState().get("route"));
        host.schedulerApplyStateCommit(invalidations);
    }

    NovaEventEnvelope enqueue(String source, String eventName, List<Object> args) {
        if (eventName == null || !eventName.startsWith("@") || eventName.length() <= 1) {
            return null;
        }
        List<Object> eventArgs = args == null ? Collections.emptyList() : new ArrayList<>(args);
        NovaEventEnvelope envelope = new NovaEventEnvelope(nextSequence++, source, eventName, eventArgs);
        queue.add(envelope);
        return envelope;
    }

    public void drain() {
        if (draining) {
            return;
        }
        draining = true;
        try {
            while (!queue.isEmpty()) {
                step();
            }
        } finally {
            draining = false;
        }
    }

    private void step() {
        NovaEventEnvelope event = queue.remove(0);
        host.schedulerBeforeEvent(event.name, event.args);
        Map<String, Object> beforeState = new LinkedHashMap<>(host.schedulerState());
        Object beforeRoute = host.schedulerCloneRoute(host.schedulerState().get("route"));
        Set<String> invalidations = applyTransitionCommit(event, beforeState);
        host.schedulerReconcileRouteBackStack(beforeRoute, host.schedulerState().get("route"));
        host.schedulerApplyStateCommit(invalidations);
        host.schedulerAfterEvent(event.name, event.args);
    }

    private Set<String> applyTransitionCommit(NovaEventEnvelope event, Map<String, Object> beforeState) {
        Map<String, Object> nextState = new LinkedHashMap<>(beforeState);
        for (NovaTransition transition : host.schedulerTransitions()) {
            if (!transition.eventName.equals(event.name)) {
                continue;
            }
            Map<String, Object> payload = new LinkedHashMap<>();
            for (int index = 0; index < transition.params.size(); index++) {
                payload.put(transition.params.get(index), index < event.args.size() ? event.args.get(index) : null);
            }
            nextState.put(transition.stateName, host.schedulerEvaluate(transition.expression, beforeState, payload));
        }
        if (host.schedulerHasRouteState()) {
            nextState.put("route", host.schedulerRouteValueForShape(nextState.get("route"), host.schedulerRoutePatterns()));
        }
        Set<String> changed = changedStates(beforeState, nextState);
        host.schedulerState().clear();
        host.schedulerState().putAll(nextState);
        return changed;
    }

    private Set<String> changedStates(Map<String, Object> beforeState, Map<String, Object> afterState) {
        Set<String> changed = new LinkedHashSet<>();
        for (String name : host.schedulerStateNames()) {
            if (name.isEmpty()) {
                continue;
            }
            Object before = beforeState.get(name);
            Object after = afterState.get(name);
            if (before == null ? after != null : !before.equals(after)) {
                changed.add(name);
            }
        }
        return changed;
    }
}

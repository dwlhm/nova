package nova.scheduler;

import java.util.ArrayList;
import java.util.Collections;
import java.util.IdentityHashMap;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Nova production scheduler for Android/Java runtimes (ADR-002).
 */
public final class NovaScheduler {
    public static final class TransitionError {
        public final String source;
        public final String phase;
        public final String event;
        public final String state;
        public final String message;

        public TransitionError(String source, String phase, String event, String state, String message) {
            this.source = source == null ? "" : source;
            this.phase = phase == null ? "" : phase;
            this.event = event == null ? "" : event;
            this.state = state == null ? "" : state;
            this.message = message == null ? "" : message;
        }
    }

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

        default String schedulerValidateEvent(String source, String eventName, List<Object> args) {
            return null;
        }

        default void schedulerOnTransitionError(String eventName, List<Object> args, List<TransitionError> errors) {}

        default void schedulerRunLifecycle(String phase, String eventName, List<Object> args) {}
    }

    private static final class CommitPlan {
        private final Map<String, Object> nextState;
        private final Set<String> invalidations;
        private final List<TransitionError> errors;

        private CommitPlan(Map<String, Object> nextState, Set<String> invalidations, List<TransitionError> errors) {
            this.nextState = nextState;
            this.invalidations = invalidations;
            this.errors = errors;
        }

        private boolean ok() {
            return errors.isEmpty();
        }
    }

    private final Host host;
    private final List<NovaEventEnvelope> queue = new ArrayList<>();
    private final List<TransitionError> errors = new ArrayList<>();
    private long nextSequence = 1L;
    private boolean draining = false;

    public NovaScheduler(Host host) {
        this.host = host;
    }

    public List<TransitionError> getErrors() {
        return Collections.unmodifiableList(errors);
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
        if (!isSchedulerEvent(eventName)) {
            return;
        }
        List<Object> eventArgs = args == null ? Collections.emptyList() : new ArrayList<>(args);
        NovaEventEnvelope event = new NovaEventEnvelope(nextSequence++, "hydrate", eventName, eventArgs);
        Map<String, Object> beforeState = new LinkedHashMap<>(host.schedulerState());
        Object beforeRoute = host.schedulerCloneRoute(host.schedulerState().get("route"));
        CommitPlan plan = planTransitionCommit(event, beforeState);
        if (!plan.ok()) {
            errors.addAll(plan.errors);
            return;
        }
        applyCommitPlan(plan, beforeRoute);
    }

    NovaEventEnvelope enqueue(String source, String eventName, List<Object> args) {
        if (!isSchedulerEvent(eventName)) {
            recordError("enqueue", eventName, "scheduler event name must start with @");
            return null;
        }
        List<Object> eventArgs = args == null ? Collections.emptyList() : new ArrayList<>(args);
        String validationError = host.schedulerValidateEvent(source, eventName, eventArgs);
        if (validationError != null) {
            recordError("enqueue", eventName, validationError);
            return null;
        }
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

    public void runLifecycle(String phase, String eventName, List<Object> args) {
        List<Object> eventArgs = args == null ? Collections.emptyList() : new ArrayList<>(args);
        host.schedulerRunLifecycle(phase == null ? "" : phase, eventName == null ? "" : eventName, eventArgs);
        drain();
    }

    private void step() {
        NovaEventEnvelope event = queue.remove(0);
        host.schedulerBeforeEvent(event.name, event.args);
        Map<String, Object> beforeState = new LinkedHashMap<>(host.schedulerState());
        Object beforeRoute = host.schedulerCloneRoute(host.schedulerState().get("route"));
        CommitPlan plan = planTransitionCommit(event, beforeState);
        if (!plan.ok()) {
            errors.addAll(plan.errors);
            host.schedulerOnTransitionError(event.name, event.args, plan.errors);
            return;
        }
        applyCommitPlan(plan, beforeRoute);
        host.schedulerAfterEvent(event.name, event.args);
    }

    private void applyCommitPlan(CommitPlan plan, Object beforeRoute) {
        host.schedulerState().clear();
        host.schedulerState().putAll(plan.nextState);
        host.schedulerReconcileRouteBackStack(beforeRoute, host.schedulerState().get("route"));
        host.schedulerApplyStateCommit(plan.invalidations);
    }

    private CommitPlan planTransitionCommit(NovaEventEnvelope event, Map<String, Object> beforeState) {
        Map<String, Object> nextState = new LinkedHashMap<>(beforeState);
        List<TransitionError> transitionErrors = new ArrayList<>();
        for (NovaTransition transition : host.schedulerTransitions()) {
            if (!transition.eventName.equals(event.name)) {
                continue;
            }
            Map<String, Object> payload = payloadForTransition(transition, event.args);
            try {
                nextState.put(transition.stateName, host.schedulerEvaluate(transition.expression, beforeState, payload));
            } catch (RuntimeException error) {
                String message = error.getMessage() == null ? String.valueOf(error) : error.getMessage();
                transitionErrors.add(new TransitionError("", "transition", event.name, transition.stateName, message));
            }
        }
        if (!transitionErrors.isEmpty()) {
            return new CommitPlan(beforeState, Collections.emptySet(), transitionErrors);
        }
        if (host.schedulerHasRouteState()) {
            nextState.put("route", host.schedulerRouteValueForShape(nextState.get("route"), host.schedulerRoutePatterns()));
        }
        Set<String> invalidations = changedStates(beforeState, nextState);
        return new CommitPlan(nextState, invalidations, Collections.emptyList());
    }

    private static Map<String, Object> payloadForTransition(NovaTransition transition, List<Object> args) {
        Map<String, Object> payload = new LinkedHashMap<>();
        List<Object> eventArgs = args == null ? Collections.emptyList() : args;
        for (int index = 0; index < transition.params.size(); index++) {
            payload.put(transition.params.get(index), index < eventArgs.size() ? eventArgs.get(index) : null);
        }
        return payload;
    }

    private Set<String> changedStates(Map<String, Object> beforeState, Map<String, Object> afterState) {
        Set<String> changed = new LinkedHashSet<>();
        for (String name : host.schedulerStateNames()) {
            if (name.isEmpty()) {
                continue;
            }
            Object before = beforeState.get(name);
            Object after = afterState.get(name);
            if (!sameValue(before, after)) {
                changed.add(name);
            }
        }
        return changed;
    }

    private void recordError(String phase, String eventName, String message) {
        errors.add(new TransitionError("", phase, eventName == null ? "" : eventName, "", message));
    }

    public static boolean isSchedulerEvent(String eventName) {
        return eventName != null && eventName.startsWith("@") && eventName.length() > 1;
    }

    public static boolean isSerializableData(Object value) {
        return isSerializableData(value, Collections.newSetFromMap(new IdentityHashMap<>()));
    }

    private static boolean isSerializableData(Object value, Set<Object> visited) {
        if (value == null) {
            return true;
        }
        if (value instanceof String || value instanceof Boolean) {
            return true;
        }
        if (value instanceof Number number) {
            double numeric = number.doubleValue();
            return !Double.isNaN(numeric) && !Double.isInfinite(numeric);
        }
        if (!(value instanceof Map<?, ?> || value instanceof List<?>)) {
            return false;
        }
        if (visited.contains(value)) {
            return false;
        }
        visited.add(value);
        if (value instanceof List<?> list) {
            for (Object item : list) {
                if (!isSerializableData(item, visited)) {
                    visited.remove(value);
                    return false;
                }
            }
            visited.remove(value);
            return true;
        }
        Map<?, ?> map = (Map<?, ?>) value;
        for (Map.Entry<?, ?> entry : map.entrySet()) {
            if (!(entry.getKey() instanceof String)) {
                visited.remove(value);
                return false;
            }
            if (!isSerializableData(entry.getValue(), visited)) {
                visited.remove(value);
                return false;
            }
        }
        visited.remove(value);
        return true;
    }

    private static boolean sameValue(Object before, Object after) {
        if (before == after) {
            return true;
        }
        if (before == null || after == null) {
            return false;
        }
        if (before instanceof Number beforeNumber && after instanceof Number afterNumber) {
            return beforeNumber.doubleValue() == afterNumber.doubleValue();
        }
        if (before instanceof Map<?, ?> beforeMap && after instanceof Map<?, ?> afterMap) {
            return mapsDeepEqual(beforeMap, afterMap);
        }
        if (before instanceof List<?> beforeList && after instanceof List<?> afterList) {
            return listsDeepEqual(beforeList, afterList);
        }
        return before.equals(after);
    }

    private static boolean mapsDeepEqual(Map<?, ?> left, Map<?, ?> right) {
        if (left.size() != right.size()) {
            return false;
        }
        for (Map.Entry<?, ?> entry : left.entrySet()) {
            if (!right.containsKey(entry.getKey())) {
                return false;
            }
            if (!sameValue(entry.getValue(), right.get(entry.getKey()))) {
                return false;
            }
        }
        return true;
    }

    private static boolean listsDeepEqual(List<?> left, List<?> right) {
        if (left.size() != right.size()) {
            return false;
        }
        for (int index = 0; index < left.size(); index++) {
            if (!sameValue(left.get(index), right.get(index))) {
                return false;
            }
        }
        return true;
    }
}

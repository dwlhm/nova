package nova.scheduler;

import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

public class NovaSchedulerTest {
    @Test
    void dispatchAppliesMatchingTransition() {
        RecordingHost host = new RecordingHost();
        host.state.put("count", 1);
        host.transitions.add(new NovaTransition("count", "@increment", List.of(), "count + 1"));
        host.stateNames.add("count");

        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.dispatch("renderer", "@increment", List.of());

        assertEquals(2, host.state.get("count"));
        assertTrue(host.committed.contains("count"));
        assertEquals(1, host.reconcileCalls);
    }

    @Test
    void dispatchRunsBeforeAndAfterHooksAroundTransition() {
        RecordingHost host = new RecordingHost();
        host.state.put("count", 1);
        host.transitions.add(new NovaTransition("count", "@increment", List.of(), "count + 1"));
        host.stateNames.add("count");

        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.dispatch("renderer", "@increment", List.of());

        assertEquals(List.of("before:@increment", "after:@increment"), host.hookTrace);
    }

    @Test
    void commitTransitionAppliesWithoutLifecycleHooks() {
        RecordingHost host = new RecordingHost();
        host.state.put("count", 1);
        host.transitions.add(new NovaTransition("count", "@hydrate", List.of(), "9"));
        host.stateNames.add("count");

        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.commitTransition("@hydrate", List.of());

        assertEquals(9, host.state.get("count"));
        assertEquals(List.of(), host.hookTrace);
    }

    @Test
    void drainRunsAfterHooksForEnqueuedLifecycleEvents() {
        RecordingHost host = new RecordingHost();
        host.state.put("count", 0);
        host.transitions.add(new NovaTransition("count", "@boot", List.of(), "1"));
        host.stateNames.add("count");

        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.enqueueLifecycle("app", "@boot", List.of());
        scheduler.drain();

        assertEquals(List.of("before:@boot", "after:@boot"), host.hookTrace);
        assertEquals(1, host.state.get("count"));
    }

    @Test
    void ignoresInvalidEventNames() {
        RecordingHost host = new RecordingHost();
        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.dispatch("renderer", "increment", List.of());
        assertEquals(0, host.committed.size());
        assertEquals(1, scheduler.getErrors().size());
    }

    @Test
    void rejectsUndeclaredEventsWhenValidationEnabled() {
        RecordingHost host = new RecordingHost();
        host.validateEvents = true;
        host.allowedEvents.put("@increment", List.of("renderer"));

        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.dispatch("renderer", "@unknown", List.of());

        assertEquals(1, scheduler.getErrors().size());
        assertTrue(scheduler.getErrors().get(0).message.contains("undeclared"));
    }

    @Test
    void rejectsDisallowedEmittersWhenValidationEnabled() {
        RecordingHost host = new RecordingHost();
        host.validateEvents = true;
        host.allowedEvents.put("@increment", List.of("renderer"));

        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.dispatch("lifecycle", "@increment", List.of());

        assertEquals(1, scheduler.getErrors().size());
        assertTrue(scheduler.getErrors().get(0).message.contains("cannot emit"));
    }

    @Test
    void transitionErrorsDoNotCommitStateOrAfterHook() {
        RecordingHost host = new RecordingHost();
        host.state.put("count", 1);
        host.transitions.add(new NovaTransition("count", "@boom", List.of(), "count + 1"));
        host.stateNames.add("count");

        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.dispatch("renderer", "@boom", List.of());

        assertEquals(1, host.state.get("count"));
        assertEquals(List.of("before:@boom", "error:@boom"), host.hookTrace);
        assertTrue(host.committed.isEmpty());
        assertEquals(1, scheduler.getErrors().size());
        assertEquals("transition", scheduler.getErrors().get(0).phase);
    }

    @Test
    void commitTransitionSkipsCommitOnTransitionError() {
        RecordingHost host = new RecordingHost();
        host.state.put("count", 1);
        host.transitions.add(new NovaTransition("count", "@hydrate", List.of(), "count + 1"));
        host.stateNames.add("count");

        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.commitTransition("@hydrate", List.of());

        assertEquals(1, host.state.get("count"));
        assertEquals(1, scheduler.getErrors().size());
    }

    @Test
    void detectsDeepStateChanges() {
        RecordingHost host = new RecordingHost();
        Map<String, Object> before = new LinkedHashMap<>();
        before.put("nested", Map.of("value", 1));
        host.state.put("record", before);
        host.transitions.add(new NovaTransition("record", "@update", List.of(), "next"));
        host.stateNames.add("record");

        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.dispatch("renderer", "@update", List.of());

        assertTrue(host.committed.contains("record"));
    }

    @Test
    void runLifecycleDelegatesToHost() {
        RecordingHost host = new RecordingHost();
        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.runLifecycle("dispose", "", List.of());
        assertEquals(List.of("lifecycle:dispose"), host.lifecycleTrace);
    }

    @Test
    void rejectsNonSerializablePayloadsWhenValidationEnabled() {
        RecordingHost host = new RecordingHost();
        host.validateEvents = true;
        host.allowedEvents.put("@increment", List.of("renderer"));

        NovaScheduler scheduler = new NovaScheduler(host);
        Object cyclic = new Object[] { null };
        ((Object[]) cyclic)[0] = cyclic;
        scheduler.dispatch("renderer", "@increment", Collections.singletonList(cyclic));

        assertEquals(1, scheduler.getErrors().size());
        assertTrue(scheduler.getErrors().get(0).message.contains("serializable"));
    }

    private static final class RecordingHost implements NovaScheduler.Host {
        private final Map<String, Object> state = new LinkedHashMap<>();
        private final List<NovaTransition> transitions = new ArrayList<>();
        private final List<String> stateNames = new ArrayList<>();
        private final Set<String> committed = new LinkedHashSet<>();
        private final List<String> hookTrace = new ArrayList<>();
        private final List<String> lifecycleTrace = new ArrayList<>();
        private final Map<String, List<String>> allowedEvents = new LinkedHashMap<>();
        private boolean validateEvents = false;
        private int reconcileCalls = 0;

        @Override
        public Map<String, Object> schedulerState() {
            return state;
        }

        @Override
        public List<NovaTransition> schedulerTransitions() {
            return transitions;
        }

        @Override
        public List<String> schedulerRoutePatterns() {
            return List.of();
        }

        @Override
        public List<String> schedulerStateNames() {
            return stateNames;
        }

        @Override
        public boolean schedulerHasRouteState() {
            return false;
        }

        @Override
        public Object schedulerCloneRoute(Object value) {
            return value;
        }

        @Override
        public Object schedulerRouteValueForShape(Object value, List<String> patterns) {
            return value;
        }

        @Override
        public Object schedulerEvaluate(String expression, Map<String, Object> current, Map<String, Object> payload) {
            if ("count + 1".equals(expression)) {
                return ((Number) current.get("count")).intValue() + 1;
            }
            if ("next".equals(expression)) {
                Map<String, Object> next = new LinkedHashMap<>();
                next.put("value", 2);
                return next;
            }
            throw new IllegalArgumentException("unsupported expression: " + expression);
        }

        @Override
        public void schedulerReconcileRouteBackStack(Object beforeRoute, Object afterRoute) {
            reconcileCalls++;
        }

        @Override
        public void schedulerApplyStateCommit(Set<String> invalidations) {
            committed.addAll(invalidations);
        }

        @Override
        public void schedulerBeforeEvent(String eventName, List<Object> args) {
            hookTrace.add("before:" + eventName);
        }

        @Override
        public void schedulerAfterEvent(String eventName, List<Object> args) {
            hookTrace.add("after:" + eventName);
        }

        @Override
        public String schedulerValidateEvent(String source, String eventName, List<Object> args) {
            if (!validateEvents) {
                return null;
            }
            List<String> emitters = allowedEvents.get(eventName);
            if (emitters == null) {
                return "undeclared scheduler event " + eventName;
            }
            if (!emitters.isEmpty() && !emitters.contains(source)) {
                return source + " cannot emit scheduler event " + eventName;
            }
            return null;
        }

        @Override
        public void schedulerOnTransitionError(String eventName, List<Object> args, List<NovaScheduler.TransitionError> errors) {
            hookTrace.add("error:" + eventName);
        }

        @Override
        public void schedulerRunLifecycle(String phase, String eventName, List<Object> args) {
            lifecycleTrace.add("lifecycle:" + phase);
        }
    }
}

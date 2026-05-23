package nova.scheduler;

import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.Arrays;
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
    void ignoresInvalidEventNames() {
        RecordingHost host = new RecordingHost();
        NovaScheduler scheduler = new NovaScheduler(host);
        scheduler.dispatch("renderer", "increment", List.of());
        assertEquals(0, host.committed.size());
    }

    private static final class RecordingHost implements NovaScheduler.Host {
        private final Map<String, Object> state = new LinkedHashMap<>();
        private final List<NovaTransition> transitions = new ArrayList<>();
        private final List<String> stateNames = new ArrayList<>();
        private final Set<String> committed = new LinkedHashSet<>();
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
    }
}

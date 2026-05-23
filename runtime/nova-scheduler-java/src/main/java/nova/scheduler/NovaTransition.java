package nova.scheduler;

import java.util.List;

public final class NovaTransition {
    public final String stateName;
    public final String eventName;
    public final List<String> params;
    public final String expression;

    public NovaTransition(String stateName, String eventName, List<String> params, String expression) {
        this.stateName = stateName;
        this.eventName = eventName;
        this.params = params;
        this.expression = expression;
    }
}

package nova.scheduler;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

final class NovaEventEnvelope {
    final long sequence;
    final String source;
    final String name;
    final List<Object> args;

    NovaEventEnvelope(long sequence, String source, String name, List<Object> args) {
        this.sequence = sequence;
        this.source = source;
        this.name = name;
        this.args = args == null ? Collections.emptyList() : new ArrayList<>(args);
    }
}

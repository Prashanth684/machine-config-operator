---
title: Configuration Drift Detection
status: Accepted
date: 2019-03-01
affected_components:
  - machine-config-daemon
---

# ADR 0003: Configuration Drift Detection

## Status

**Accepted**

## Context

Nodes in an OpenShift cluster may experience configuration drift where the actual state diverges from the desired state defined in MachineConfigs. Drift can occur due to:
- Manual changes by operators (SSH access, direct file edits)
- System processes modifying files
- Failed config applications
- External configuration management tools

Without drift detection, the cluster state becomes unpredictable and debugging becomes difficult.

## Decision

Implement **periodic configuration drift detection** in the machine-config-daemon. MCD continuously monitors node state and reconciles drift back to the desired configuration.

## Rationale

- ✅ **Self-Healing**: Automatically corrects drift without manual intervention
- ✅ **Predictable State**: Ensures nodes always match desired configuration
- ✅ **Early Detection**: Catches configuration issues before they cause failures
- ✅ **SSH Safety**: Detects and alerts on manual changes (improves cluster hygiene)
- ✅ **Audit Trail**: Node annotations provide visibility into drift events

## Alternatives Considered

### Alternative 1: No Drift Detection
- **Pro**: Simpler implementation
- **Pro**: Lower CPU/memory overhead
- **Con**: Manual changes persist indefinitely
- **Con**: Cluster state becomes unpredictable
- **Con**: Debugging becomes very difficult
- **Why not chosen**: Unacceptable for production clusters (predictability is critical)

### Alternative 2: Manual Reconciliation (on-demand)
- **Pro**: Operator controls when to reconcile
- **Pro**: Lower overhead
- **Con**: Requires operator awareness of drift
- **Con**: Drift may persist for long periods
- **Con**: No automatic correction
- **Why not chosen**: Doesn't scale across large clusters with many nodes

### Alternative 3: Event-Based Detection (inotify)
- **Pro**: Immediate detection of file changes
- **Pro**: Low overhead when no changes occur
- **Con**: Complex to implement (watch all managed files)
- **Con**: inotify limits can be exceeded
- **Con**: Doesn't detect OS-level drift (rpm-ostree)
- **Why not chosen**: Periodic check is simpler and catches all drift types

## Consequences

**Positive**:
- Nodes self-heal from manual changes
- Cluster state remains consistent and predictable
- Operators warned about SSH access (annotation)
- Easier debugging (drift events visible in node status)

**Negative**:
- Manual emergency fixes may be overwritten (operators must use MachineConfigs)
- Periodic reconciliation consumes CPU/memory
- May conflict with external config management tools

## Affected Components

- **machine-config-daemon**: Implements drift detection and reconciliation loop

## Mitigation

- **Emergency fixes**: Document how to pause pools for emergency changes
- **Overhead**: Run drift check every 5 minutes (configurable), low-cost operations
- **External tools**: Document that external config management is unsupported (use MachineConfigs instead)
- **SSH access**: Annotate nodes with `machineconfiguration.openshift.io/ssh=accessed` (visibility, not blocking)

## Implementation Details

### Drift Detection Loop

MCD runs periodic reconciliation (every 5 minutes by default):

```go
1. Read desiredConfig from node annotation
2. Hash current node state (files, systemd units, OS version)
3. Compare with stored hash of desiredConfig
4. If mismatch:
   a. Log drift event
   b. Reapply configuration
   c. Update currentConfig annotation
   d. Update state annotation (Degraded if failed)
```

### State Tracking

MCD maintains state via node annotations:

| Annotation | Purpose |
|------------|---------|
| `machineconfiguration.openshift.io/currentConfig` | Current rendered config hash |
| `machineconfiguration.openshift.io/desiredConfig` | Desired rendered config hash |
| `machineconfiguration.openshift.io/state` | Update state (Done, Working, Degraded) |
| `machineconfiguration.openshift.io/reason` | Reason for current state |

### Drift Detection Mechanisms

**File drift**:
```bash
# MCD verifies each managed file matches desired content
sha256sum /etc/sysctl.d/99-custom.conf
# If mismatch, rewrite file
```

**Systemd unit drift**:
```bash
# MCD verifies systemd units match desired state
systemctl cat custom.service
# If mismatch, reload unit
systemctl daemon-reload
```

**OS version drift**:
```bash
# MCD verifies OS matches desiredConfig
rpm-ostree status
# If mismatch, trigger OS update
```

### SSH Access Detection

MCD detects SSH access and annotates node:

```bash
# Check for SSH sessions
who | grep -v console
# If SSH detected, annotate node
oc annotate node/<node> machineconfiguration.openshift.io/ssh=accessed
```

**Purpose**: Visibility into manual access (not enforcement)

### Pausing for Emergency Changes

Operators can pause pools to make emergency changes:

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfigPool
metadata:
  name: worker
spec:
  paused: true  # Stop reconciliation
```

**Workflow**:
1. Pause pool
2. Make emergency changes via SSH
3. Test changes
4. Codify changes in MachineConfig
5. Unpause pool (MCD will reconcile to MachineConfig)

## Performance Considerations

**Reconciliation frequency**: Every 5 minutes (configurable via environment variable)

**Overhead per check**:
- File hashing: ~10ms per file (minimal CPU)
- Systemd state: ~50ms (systemctl commands)
- rpm-ostree status: ~100ms

**Total overhead**: <1% CPU on typical nodes with 10-20 managed files

## Observability

**Node annotations**: Visible in `oc describe node/<node>`

**MCD logs**: Drift events logged:
```
MCD: Detected config drift on /etc/sysctl.d/99-custom.conf
MCD: Reapplying configuration
MCD: Config reconciliation complete
```

**Metrics**: `mcd_drift_events_total` (Prometheus metric)

## References

- **Related**: [MachineConfig](../domain/machineconfig.md)
- **Related**: [MachineConfigDaemon](../architecture/components.md#3-machine-config-daemon-mcd)
- **Related**: [MachineConfigPool](../domain/machineconfigpool.md)

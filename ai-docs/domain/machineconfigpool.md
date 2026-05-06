# MachineConfigPool

**API Group**: `machineconfiguration.openshift.io/v1`  
**Kind**: `MachineConfigPool`  
**Scope**: Cluster (non-namespaced)

## Purpose

Groups nodes by role and coordinates rolling updates when MachineConfigs change. Controls update strategy (max unavailable, pause), tracks update progress, and reports pool health. Default pools are `master` and `worker`; custom pools enable staged rollouts.

## Key Fields

### Spec

| Field | Type | Description |
|-------|------|-------------|
| `machineConfigSelector` | LabelSelector | Selects MachineConfigs to merge for this pool |
| `nodeSelector` | LabelSelector | Selects nodes that belong to this pool |
| `paused` | bool | If true, stop updates (no new rendering, no node updates) |
| `maxUnavailable` | IntOrString | Max nodes updating simultaneously (number or %) |
| `configuration.name` | string | Deprecated: rendered MachineConfig name |
| `configuration.source` | []ObjectReference | MachineConfigs included in rendered config |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `observedGeneration` | int64 | Last processed spec generation |
| `configuration.name` | string | Current rendered MachineConfig name (hash-based) |
| `configuration.source` | []ObjectReference | MachineConfigs in current render |
| `machineCount` | int32 | Total nodes in pool |
| `updatedMachineCount` | int32 | Nodes with current config |
| `readyMachineCount` | int32 | Nodes ready and updated |
| `unavailableMachineCount` | int32 | Nodes updating or degraded |
| `degradedMachineCount` | int32 | Nodes failing to update |
| `conditions` | []Condition | Pool health (Updated, Updating, Degraded, RenderDegraded) |

## Lifecycle

1. **Pool Creation**: Default pools (`master`, `worker`) created at install; custom pools via CR
2. **Config Rendering**: MCC watches MachineConfigs matching `machineConfigSelector`, merges, renders final config
3. **Update Coordination**: MCC updates nodes respecting `maxUnavailable`
4. **Progress Tracking**: MCC updates status as nodes transition through update
5. **Completion**: Pool marked `Updated=true` when all nodes have current config

## Update Coordination

### Update Flow

```
MachineConfig changes
  ↓
MCC renders new config (status.configuration.name = rendered-master-abc123)
  ↓
MCC identifies nodes needing update (currentConfig ≠ desiredConfig)
  ↓
MCC cordons/drains nodes (respects maxUnavailable)
  ↓
MCD applies config on node (may reboot)
  ↓
Node uncordoned, marked updated
  ↓
Repeat until all nodes updated
```

### maxUnavailable

Controls update velocity:
- **Number** (e.g., `2`): Max 2 nodes updating simultaneously
- **Percentage** (e.g., `10%`): Max 10% of pool nodes updating
- **Default**: `1` for master, `1` for worker (can be customized)

**Example**: Pool with 10 nodes, `maxUnavailable: 2`:
- Update 2 nodes
- Wait for both to complete
- Update next 2 nodes
- ...

## Condition Types

| Type | True Means | False Means |
|------|------------|-------------|
| `Updated` | All nodes have current config | Some nodes outdated |
| `Updating` | Update in progress | Pool idle |
| `Degraded` | Some nodes failing to update | All nodes healthy or updating |
| `NodeDegraded` | Some nodes degraded | All nodes healthy |
| `RenderDegraded` | Failed to render config | Render succeeded |

## Custom Pools

Create custom pools for staged rollouts (e.g., canary nodes, edge nodes):

### Example: Canary Pool

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfigPool
metadata:
  name: worker-canary
spec:
  machineConfigSelector:
    matchExpressions:
    - key: machineconfiguration.openshift.io/role
      operator: In
      values: [worker, worker-canary]
  nodeSelector:
    matchLabels:
      node-role.kubernetes.io/worker-canary: ""
  paused: false
  maxUnavailable: 1
```

**Node labeling**:
```bash
oc label node <node-name> node-role.kubernetes.io/worker-canary=""
```

**Update flow**:
1. Apply MachineConfig with `role: worker-canary`
2. `worker-canary` pool updates first
3. Verify health
4. Apply MachineConfig with `role: worker`
5. `worker` pool updates

## Pausing Updates

Pause pool to stop updates (e.g., during maintenance):

```yaml
spec:
  paused: true
```

**Effect**:
- MCC stops rendering new configs for this pool
- No nodes updated
- Pool status frozen

**Resume**: Set `paused: false`

## Component-Specific Behavior

### Rendered Config Naming

Rendered MachineConfig name format: `rendered-<pool>-<hash>`

**Example**: `rendered-worker-abc123def456`

**Hash**: SHA-256 of merged Ignition config (first 12 chars)

### Node Annotations

Nodes annotated with:
- `machineconfiguration.openshift.io/currentConfig`: Current rendered config
- `machineconfiguration.openshift.io/desiredConfig`: Desired rendered config
- `machineconfiguration.openshift.io/state`: Update state (Done, Working, Degraded)

### Update Safety

MCC ensures:
- Never exceed `maxUnavailable`
- Drain nodes before update (evict pods)
- Wait for node ready before moving to next node
- Mark pool Degraded if node fails to update after retries

## Examples

### View Pool Status

```bash
oc get machineconfigpool
oc describe machineconfigpool/worker
```

**Output**:
```
NAME     CONFIG                             UPDATED   UPDATING   DEGRADED   MACHINECOUNT   READYMACHINECOUNT   UPDATEDMACHINECOUNT   DEGRADEDMACHINECOUNT   AGE
master   rendered-master-abc123             True      False      False      3              3                   3                     0                      10d
worker   rendered-worker-def456             False     True       False      5              3                   3                     0                      10d
```

### Increase Update Velocity

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfigPool
metadata:
  name: worker
spec:
  maxUnavailable: 3  # Update 3 nodes simultaneously
```

## References

**Tier 1 Patterns**: [Status Conditions](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/status.md) | [Controller Runtime](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/controllers.md)

**Related CRDs**: [MachineConfig](machineconfig.md) | [KubeletConfig](kubeletconfig.md)

**Architecture**: [Components](../architecture/components.md)

**Decisions**: [ADR-0003 Config Drift](../decisions/adr-0003-config-drift.md)

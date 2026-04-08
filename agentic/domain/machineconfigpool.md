# MachineConfigPool

**API Group**: `machineconfiguration.openshift.io/v1`  
**Kind**: `MachineConfigPool`  
**Scope**: Cluster

## Purpose

MachineConfigPool (MCP) groups machines and controls how MachineConfig updates roll out to them. Each pool has a rendered MachineConfig target, and the UpdateController coordinates upgrading machines in the pool to that target.

**Key Principle**: Pools enable independent update control for different node types (master, worker, custom).

## Spec Structure

```go
type MachineConfigPoolSpec struct {
    MachineConfigSelector *metav1.LabelSelector  // Selects MachineConfigs to merge
    NodeSelector          *metav1.LabelSelector  // Selects nodes in this pool
    Paused                bool                    // Stop updates if true
    MaxUnavailable        *intstr.IntOrString    // Max nodes updating simultaneously
    Configuration         MachineConfigReference // Desired rendered config
}
```

## Key Concepts

### Pool Selection

**NodeSelector** determines which nodes belong to this pool:
```yaml
nodeSelector:
  matchLabels:
    node-role.kubernetes.io/worker: ""
```

**MachineConfigSelector** determines which MachineConfigs to merge:
```yaml
machineConfigSelector:
  matchLabels:
    machineconfiguration.openshift.io/role: worker
```

### Rendered Configuration

The **RenderController** merges all MachineConfigs matching the pool's selector:
- Sorts configs lexicographically by name (`00-` → `99-`)
- Merges Ignition configs (later configs override earlier)
- Generates rendered config: `rendered-worker-<hash>`
- Updates `spec.configuration.name` with rendered config name

### Update Coordination

The **UpdateController** manages rollout:
1. Compares node's `currentConfig` annotation vs pool's `configuration.name`
2. Respects `maxUnavailable` to limit concurrent updates
3. Coordinates with MCD via node annotations:
   - `machineconfiguration.openshift.io/desiredConfig`: Target config
   - `machineconfiguration.openshift.io/currentConfig`: Active config
   - `machineconfiguration.openshift.io/state`: Done/Working/Degraded

### Status Tracking

```go
type MachineConfigPoolStatus struct {
    Configuration              MachineConfigReference
    MachineCount               int32  // Total nodes in pool
    UpdatedMachineCount        int32  // Nodes at desired config
    ReadyMachineCount          int32  // Ready nodes
    UnavailableMachineCount    int32  // Updating or not ready
    DegradedMachineCount       int32  // Degraded nodes
    Conditions                 []metav1.Condition
}
```

**Conditions** follow standard Available/Progressing/Degraded semantics (see Tier 1 operator patterns).

## Default Pools

OpenShift creates two default pools:

- **master**: Control plane nodes
  - NodeSelector: `node-role.kubernetes.io/master`
  - Updates respect etcd quorum (via PodDisruptionBudgets)

- **worker**: Worker nodes
  - NodeSelector: `node-role.kubernetes.io/worker`
  - Default `maxUnavailable: 1`

## Custom Pools

Users can create custom pools for specialized workloads:
```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfigPool
metadata:
  name: gpu-workers
spec:
  machineConfigSelector:
    matchExpressions:
    - key: machineconfiguration.openshift.io/role
      operator: In
      values: [worker, gpu-workers]
  nodeSelector:
    matchLabels:
      node-role.kubernetes.io/gpu-worker: ""
  maxUnavailable: 2
```

**Pattern**: Custom pool inherits worker configs plus pool-specific configs.

## Update Control

### Pausing Updates

Set `paused: true` to stop updates:
```bash
oc patch mcp/worker --type merge --patch '{"spec":{"paused":true}}'
```

Useful for:
- Maintenance windows
- Investigating degraded nodes
- Coordinating with other changes

### MaxUnavailable

Controls update velocity:
- `maxUnavailable: 1` (default): Serial updates, safest
- `maxUnavailable: 10%`: Percentage of pool size
- `maxUnavailable: 5`: Absolute number

**Constraint**: At least 1 node always updates (cannot be 0).

**Interaction**: Respects PodDisruptionBudgets during drain.

## Monitoring Pool Status

```bash
# Check pool status
oc get mcp
oc describe mcp/worker

# Watch update progress
oc get mcp -w

# Check node status
oc get nodes -o custom-columns=NAME:.metadata.name,CURRENT:.metadata.annotations.machineconfiguration\.openshift\.io/currentConfig,DESIRED:.metadata.annotations.machineconfiguration\.openshift\.io/desiredConfig,STATE:.metadata.annotations.machineconfiguration\.openshift\.io/state
```

## Degraded States

A pool becomes Degraded when:
- MCD cannot apply config (unsupported Ignition changes)
- Config drift detected on node
- OS update fails
- Drain timeout exceeded

**Recovery**: Fix the issue or create forcefile (`/run/machine-config-daemon-force`) to force reboot.

## Related Concepts

- **MachineConfig**: The configuration content merged into the pool
- **Node**: Machines managed by the pool
- **MachineSet**: Cloud-provider machine provisioning (orthogonal to pools)

## References

- Design doc: `docs/MachineConfigController.md`
- Custom pools: `docs/custom-pools.md`
- Status conditions: See Tier 1 operator patterns

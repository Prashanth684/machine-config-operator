# ContainerRuntimeConfig

**API Group**: `machineconfiguration.openshift.io/v1`  
**Kind**: `ContainerRuntimeConfig`  
**Scope**: Cluster (non-namespaced)

## Purpose

Configures CRI-O container runtime settings for nodes in specific MachineConfigPools. ContainerRuntimeConfig CRs are processed by the containerruntime-config-controller, which generates MachineConfigs containing CRI-O configuration files. Enables declarative tuning of container runtime without manual config editing.

## Key Fields

### Spec

| Field | Type | Description |
|-------|------|-------------|
| `machineConfigPoolSelector` | LabelSelector | Selects MachineConfigPools to apply config to |
| `containerRuntimeConfig` | ContainerRuntimeConfiguration | CRI-O config object |

### ContainerRuntimeConfiguration

| Field | Type | Description |
|-------|------|-------------|
| `pidsLimit` | int64 | Max PIDs per container (default: 1024) |
| `logLevel` | string | CRI-O log level (error, warn, info, debug) |
| `logSizeMax` | Quantity | Max size of container log file |
| `overlaySize` | Quantity | Max overlay filesystem size for containers |
| `defaultRuntime` | string | Default OCI runtime (runc, crun, kata) |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `observedGeneration` | int64 | Last processed spec generation |
| `conditions` | []Condition | Processing status (Success, Failure) |

## Lifecycle

1. **Creation**: User creates ContainerRuntimeConfig targeting specific pools
2. **Processing**: containerruntime-config-controller watches ContainerRuntimeConfig
3. **MachineConfig Generation**: Controller generates `99-<pool>-generated-containerruntime` MachineConfig
4. **Pool Update**: Generated MachineConfig triggers MachineConfigPool update
5. **Node Application**: MCD applies CRI-O config, restarts CRI-O

## Component-Specific Behavior

### Generated MachineConfig

ContainerRuntimeConfig generates a MachineConfig named `99-<pool>-generated-containerruntime`.

**Contents**:
- File: `/etc/crio/crio.conf.d/01-ctrcfg-pidsLimit` (PIDs limit)
- File: `/etc/crio/crio.conf.d/01-ctrcfg-logLevel` (log level)
- File: `/etc/crio/crio.conf.d/01-ctrcfg-logSizeMax` (log size)
- File: `/etc/crio/crio.conf.d/01-ctrcfg-overlaySize` (overlay size)
- File: `/etc/crio/crio.conf.d/01-ctrcfg-default-runtime` (default runtime)

**CRI-O config structure** (TOML format):
```toml
[crio.runtime]
pids_limit = 2048
log_level = "debug"
default_runtime = "crun"

[crio.runtime.runtimes.runc]
runtime_path = "/usr/bin/runc"

[crio.runtime.runtimes.crun]
runtime_path = "/usr/bin/crun"
```

### Pool Selection

Use `machineConfigPoolSelector` to target specific pools:

```yaml
machineConfigPoolSelector:
  matchLabels:
    pools.operator.machineconfiguration.openshift.io/worker: ""
```

### Update Behavior

When ContainerRuntimeConfig changes:
1. Controller regenerates `99-<pool>-generated-containerruntime` MachineConfig
2. MachineConfigPool detects MachineConfig change
3. Pool initiates rolling update
4. MCD restarts CRI-O on each node (no reboot required)
5. Running containers continue (CRI-O graceful restart)

## Examples

### Increase PIDs Limit

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: increase-pids-limit
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  containerRuntimeConfig:
    pidsLimit: 4096
```

**Use case**: High-density workloads with many processes per container

### Enable Debug Logging

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: crio-debug-logging
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  containerRuntimeConfig:
    logLevel: debug
```

**Use case**: Troubleshooting container startup issues

### Configure Log Size

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: increase-log-size
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  containerRuntimeConfig:
    logSizeMax: 100Mi
```

**Use case**: Verbose applications requiring larger log buffers

### Configure Overlay Size

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: increase-overlay-size
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  containerRuntimeConfig:
    overlaySize: 20Gi
```

**Use case**: Workloads with large ephemeral storage needs

### Switch Default Runtime

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: use-crun
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  containerRuntimeConfig:
    defaultRuntime: crun
```

**Available runtimes**:
- `runc`: Original OCI runtime (default)
- `crun`: Fast, low-memory C runtime
- `kata`: Kata Containers (VM isolation, requires setup)

## PIDs Limit Details

### Default Behavior

- **Default PIDs limit**: 1024 PIDs per container
- **Unlimited**: Set `pidsLimit: 0` to disable limit (not recommended)

### Impact

**Too low**:
- Containers fail with "cannot fork" errors
- Java apps, databases hit limit easily

**Too high**:
- Risk of PID exhaustion on node
- Can impact node stability

**Recommendation**: Start with 2048, tune based on workload

## CRI-O Restart Safety

When MCD applies ContainerRuntimeConfig:
1. CRI-O gracefully restarts
2. Running containers continue (not restarted)
3. New containers use new config
4. Node remains schedulable (no drain)

**Exception**: Some settings require container restart to take effect (e.g., PIDs limit applies to new containers only)

## Validation

The containerruntime-config-controller validates ContainerRuntimeConfig:
- Must target at least one MachineConfigPool
- PIDs limit must be >= 0
- Log level must be valid (error, warn, info, debug)
- Quantities must be valid (logSizeMax, overlaySize)

**Status condition** reports validation results:
```yaml
status:
  conditions:
  - type: Success
    status: "True"
    reason: Success
    message: "Success"
```

## References

**Tier 1 Patterns**: [Controller Runtime](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/controllers.md)

**Related CRDs**: [MachineConfig](machineconfig.md) | [MachineConfigPool](machineconfigpool.md) | [KubeletConfig](kubeletconfig.md)

**Architecture**: [Components](../architecture/components.md)

**Upstream**: [CRI-O Configuration](https://github.com/cri-o/cri-o/blob/main/docs/crio.conf.5.md)

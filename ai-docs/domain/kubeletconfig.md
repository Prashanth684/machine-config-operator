# KubeletConfig

**API Group**: `machineconfiguration.openshift.io/v1`  
**Kind**: `KubeletConfig`  
**Scope**: Cluster (non-namespaced)

## Purpose

Configures kubelet settings for nodes in specific MachineConfigPools. KubeletConfig CRs are processed by the kubelet-config-controller, which generates MachineConfigs containing kubelet configuration files. This provides a declarative way to tune kubelet without manually editing MachineConfigs.

## Key Fields

### Spec

| Field | Type | Description |
|-------|------|-------------|
| `machineConfigPoolSelector` | LabelSelector | Selects MachineConfigPools to apply config to |
| `kubeletConfig` | KubeletConfiguration | Kubelet config object (subset of kubelet flags) |
| `tlsSecurityProfile` | TLSSecurityProfile | TLS settings for kubelet |
| `logLevel` | int32 | Kubelet log verbosity (0-10) |
| `autoSizingReserved` | bool | Enable automatic reserved resource calculation |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `observedGeneration` | int64 | Last processed spec generation |
| `conditions` | []Condition | Processing status (Success, Failure) |

## Lifecycle

1. **Creation**: User creates KubeletConfig targeting specific pools
2. **Processing**: kubelet-config-controller watches KubeletConfig
3. **MachineConfig Generation**: Controller generates `99-<pool>-generated-kubelet` MachineConfig
4. **Pool Update**: Generated MachineConfig triggers MachineConfigPool update
5. **Node Application**: MCD applies kubelet config, restarts kubelet

## Kubelet Configuration Fields

Common fields (subset of [kubelet config](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/)):

| Field | Purpose | Example |
|-------|---------|---------|
| `maxPods` | Max pods per node | `250` |
| `podsPerCore` | Max pods per CPU core | `10` |
| `systemReserved` | Reserved CPU/memory for system | `cpu: 500m, memory: 1Gi` |
| `kubeReserved` | Reserved CPU/memory for kubelet/k8s | `cpu: 500m, memory: 1Gi` |
| `evictionHard` | Hard eviction thresholds | `memory.available: 500Mi` |
| `evictionSoft` | Soft eviction thresholds | `memory.available: 1Gi` |
| `imageMinimumGCAge` | Min image age before GC | `2m` |
| `imageGCHighThresholdPercent` | Disk usage to trigger image GC | `85` |

## Component-Specific Behavior

### Generated MachineConfig

KubeletConfig generates a MachineConfig named `99-<pool>-generated-kubelet`.

**Contents**:
- File: `/etc/kubernetes/kubelet.conf` (kubelet configuration)
- File: `/etc/kubernetes/compliance-operator/kubeletconfig/<pool>` (for compliance operator)

**Example**:
```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: worker
  name: 99-worker-generated-kubelet
spec:
  config:
    ignition:
      version: 3.2.0
    storage:
      files:
      - path: /etc/kubernetes/kubelet.conf
        mode: 0644
        contents:
          source: data:,<base64-encoded-kubelet-config>
```

### Pool Selection

Use `machineConfigPoolSelector` to target specific pools:

**Label matching**:
```yaml
machineConfigPoolSelector:
  matchLabels:
    custom-kubelet: high-density
```

**Pool must have matching label**:
```bash
oc label machineconfigpool/worker custom-kubelet=high-density
```

### Update Behavior

When KubeletConfig changes:
1. Controller regenerates `99-<pool>-generated-kubelet` MachineConfig
2. MachineConfigPool detects MachineConfig change
3. Pool initiates rolling update
4. MCD restarts kubelet on each node (no reboot required)

## Examples

### Increase Max Pods

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: increase-max-pods
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  kubeletConfig:
    maxPods: 500
```

### System Reservations

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: worker-system-reserved
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  kubeletConfig:
    systemReserved:
      cpu: 1000m
      memory: 2Gi
    kubeReserved:
      cpu: 500m
      memory: 1Gi
```

### Eviction Thresholds

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: worker-eviction
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  kubeletConfig:
    evictionHard:
      memory.available: 500Mi
      nodefs.available: 10%
      imagefs.available: 15%
    evictionSoft:
      memory.available: 1Gi
      nodefs.available: 15%
    evictionSoftGracePeriod:
      memory.available: 1m30s
      nodefs.available: 1m30s
```

### Custom Pool

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: high-density-kubelet
spec:
  machineConfigPoolSelector:
    matchLabels:
      node-role.kubernetes.io/infra: ""
  kubeletConfig:
    maxPods: 1000
    podsPerCore: 50
    systemReserved:
      cpu: 2000m
      memory: 4Gi
```

## Validation

The kubelet-config-controller validates KubeletConfig:
- Must target at least one MachineConfigPool
- Kubelet config must be valid (no unknown fields)
- Resource reservations must be valid quantities

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

**Related CRDs**: [MachineConfig](machineconfig.md) | [MachineConfigPool](machineconfigpool.md) | [ContainerRuntimeConfig](containerruntimeconfig.md)

**Architecture**: [Components](../architecture/components.md)

**Upstream**: [Kubelet Configuration](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/)

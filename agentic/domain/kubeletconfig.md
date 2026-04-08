# KubeletConfig

**API Group**: `machineconfiguration.openshift.io/v1`  
**Kind**: `KubeletConfig`  
**Scope**: Cluster

## Purpose

KubeletConfig provides a high-level API for configuring kubelet settings without requiring deep knowledge of kubelet configuration file format. The KubeletConfigController watches KubeletConfig objects and generates corresponding MachineConfigs.

**Key Principle**: User-friendly abstraction that generates low-level MachineConfigs.

## Spec Structure

```go
type KubeletConfigSpec struct {
    MachineConfigPoolSelector *metav1.LabelSelector  // Target pools
    KubeletConfig              *kubeletconfigv1beta1.KubeletConfiguration
    LogLevel                   *int32
    TLSSecurityProfile         *configv1.TLSSecurityProfile
    AutoSizingReserved         *AutoSizingReservedConfiguration
}
```

## How It Works

1. **User creates KubeletConfig** with desired kubelet settings
2. **KubeletConfigController** watches for changes
3. **Controller generates MachineConfig** named like `99-<pool>-<hash>-kubelet`
4. **MachineConfig merges** with other configs in the pool
5. **MCD applies** kubelet configuration to nodes
6. **Node reboots** to activate new kubelet settings

## Common Use Cases

### Set Max Pods Per Node

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: set-max-pods
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  kubeletConfig:
    maxPods: 500
```

### Configure CPU Manager

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: cpu-manager-static
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  kubeletConfig:
    cpuManagerPolicy: static
    cpuManagerReconcilePeriod: 5s
```

### Configure Memory/CPU Reservations

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: set-node-resources
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  kubeletConfig:
    systemReserved:
      cpu: 500m
      memory: 1Gi
    kubeReserved:
      cpu: 500m
      memory: 1Gi
```

### Enable Feature Gates

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: enable-feature-gate
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/master: ""
  kubeletConfig:
    featureGates:
      RotateKubeletServerCertificate: true
```

## Generated MachineConfig

The controller generates a MachineConfig that:
- Writes `/etc/kubernetes/kubelet.conf` with merged kubelet configuration
- Updates kubelet systemd unit if needed
- Sets ownership to MCO controller

**Naming**: `99-<pool>-generated-kubelet` or `99-<pool>-<name>-kubelet`

**Owner Reference**: MachineConfig has owner reference to KubeletConfig

## Lifecycle

1. **Create**: User creates KubeletConfig
2. **Generate**: Controller creates MachineConfig
3. **Render**: MachineConfig merges into pool's rendered config
4. **Apply**: MCD applies kubelet config and reboots
5. **Delete**: Deleting KubeletConfig removes generated MachineConfig

**Restoration**: Deleting KubeletConfig restores default kubelet settings.

## Available Kubelet Options

KubeletConfig wraps Kubernetes KubeletConfiguration v1beta1. Available fields include:

- **Resource Management**: `maxPods`, `podsPerCore`, `systemReserved`, `kubeReserved`
- **CPU Manager**: `cpuManagerPolicy`, `cpuManagerReconcilePeriod`
- **Memory Manager**: `memoryManagerPolicy`, `reservedMemory`
- **Eviction Policies**: `evictionHard`, `evictionSoft`, `evictionPressureTransitionPeriod`
- **Feature Gates**: `featureGates` (enable/disable Kubernetes features)
- **Logging**: `logging` configuration
- **Container Runtime**: `containerLogMaxSize`, `containerLogMaxFiles`

See [Kubernetes KubeletConfiguration](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/) for full reference.

## Pool Targeting

Use `machineConfigPoolSelector` to target specific pools:

```yaml
# Target worker pool
machineConfigPoolSelector:
  matchLabels:
    pools.operator.machineconfiguration.openshift.io/worker: ""

# Target master pool
machineConfigPoolSelector:
  matchLabels:
    pools.operator.machineconfiguration.openshift.io/master: ""

# Target custom pool
machineConfigPoolSelector:
  matchLabels:
    pools.operator.machineconfiguration.openshift.io/gpu-worker: ""
```

## Validation

The controller validates:
- KubeletConfiguration fields against Kubernetes schema
- Pool selector matches existing pools
- Settings are compatible with OpenShift

**Invalid configs** cause the KubeletConfig to report errors in status.

## Debugging

```bash
# Check KubeletConfig status
oc get kubeletconfig
oc describe kubeletconfig/<name>

# View generated MachineConfig
oc get mc | grep kubelet
oc describe mc/99-worker-generated-kubelet

# Check kubelet config on node
oc debug node/<node-name>
chroot /host
cat /etc/kubernetes/kubelet.conf
```

## Limitations

- Requires node reboot (kubelet config changes need restart)
- Some kubelet options may be incompatible with OpenShift requirements
- Cannot override platform-required kubelet settings

## Related Concepts

- **MachineConfig**: Low-level config that KubeletConfig generates
- **ContainerRuntimeConfig**: Similar high-level API for CRI-O
- **MachineConfigPool**: Target for kubelet configuration

## References

- Design doc: `docs/KubeletConfigDesign.md`
- Kubernetes KubeletConfiguration: https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/

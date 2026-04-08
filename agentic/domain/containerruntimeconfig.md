# ContainerRuntimeConfig

**API Group**: `machineconfiguration.openshift.io/v1`  
**Kind**: `ContainerRuntimeConfig`  
**Scope**: Cluster

## Purpose

ContainerRuntimeConfig provides a high-level API for configuring CRI-O container runtime settings. The ContainerRuntimeConfigController watches these objects and generates corresponding MachineConfigs.

**Key Principle**: User-friendly abstraction for CRI-O configuration that generates low-level MachineConfigs.

## Spec Structure

```go
type ContainerRuntimeConfigSpec struct {
    MachineConfigPoolSelector  *metav1.LabelSelector
    ContainerRuntimeConfig     *ContainerRuntimeConfiguration
}

type ContainerRuntimeConfiguration struct {
    PidsLimit              *int64
    LogLevel               string
    LogSizeMax             *resource.Quantity
    OverlaySize            *resource.Quantity
    DefaultRuntime         ContainerRuntimeDefaultRuntime
}
```

## How It Works

1. **User creates ContainerRuntimeConfig** with desired CRI-O settings
2. **ContainerRuntimeConfigController** watches for changes
3. **Controller generates MachineConfig** named like `99-<pool>-<hash>-containerruntime`
4. **MachineConfig writes** `/etc/crio/crio.conf.d/` configuration
5. **MCD applies** and may reload or reboot depending on changes
6. **CRI-O picks up** new configuration

## Common Use Cases

### Set PIDs Limit

Prevents fork bombs by limiting container PIDs:

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: set-pids-limit
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  containerRuntimeConfig:
    pidsLimit: 2048
```

### Configure Log Settings

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: configure-logging
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  containerRuntimeConfig:
    logLevel: debug
    logSizeMax: 100Mi
```

### Set Overlay Size

Configure container overlay filesystem size:

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: set-overlay-size
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  containerRuntimeConfig:
    overlaySize: 50Gi
```

## Generated MachineConfig

The controller generates a MachineConfig that:
- Writes CRI-O configuration to `/etc/crio/crio.conf.d/01-ctrcfg-<name>`
- May write storage configuration to `/etc/containers/storage.conf`
- Sets owner reference to ContainerRuntimeConfig

**Naming**: `99-<pool>-generated-containerruntime` or `99-<pool>-<name>-containerruntime`

## Update Behavior

Unlike most MachineConfig changes, some CRI-O config updates avoid full reboot:

**Reload CRI-O only** (no drain/reboot):
- PIDs limit changes
- Log level changes
- Log size changes

**Requires reboot**:
- Overlay size changes (storage configuration)
- Default runtime changes

See `docs/MachineConfigDaemon.md` for rebootless update details.

## Lifecycle

1. **Create**: User creates ContainerRuntimeConfig
2. **Generate**: Controller creates MachineConfig with CRI-O config
3. **Render**: MachineConfig merges into pool's rendered config
4. **Apply**: MCD applies config (reload or reboot)
5. **Delete**: Deleting ContainerRuntimeConfig removes generated config

**Restoration**: Deleting ContainerRuntimeConfig restores default CRI-O settings.

## Available Options

### pidsLimit

Maximum PIDs per container (prevents fork bombs):
- Default: 1024
- Recommended: 2048-4096 for workloads with many processes

### logLevel

CRI-O logging verbosity:
- Values: `error`, `warn`, `info`, `debug`, `trace`
- Default: `info`
- Use `debug` for troubleshooting

### logSizeMax

Maximum size of container logs before rotation:
- Format: Kubernetes resource.Quantity (`50Mi`, `100Mi`)
- Default: 50Mi

### overlaySize

Size of container overlay filesystem:
- Format: Kubernetes resource.Quantity (`50Gi`, `100Gi`)
- Requires reboot to change

### defaultRuntime

Select container runtime:
- `crun` (default): Fast, lightweight OCI runtime
- `runc`: Traditional OCI runtime

## Pool Targeting

Target specific pools using `machineConfigPoolSelector`:

```yaml
# Target worker pool
machineConfigPoolSelector:
  matchLabels:
    pools.operator.machineconfiguration.openshift.io/worker: ""
```

## Validation

The controller validates:
- Configuration values are within acceptable ranges
- Pool selector matches existing pools
- Settings are compatible with CRI-O version

**Invalid configs** cause errors in ContainerRuntimeConfig status.

## Debugging

```bash
# Check ContainerRuntimeConfig status
oc get containerruntimeconfig
oc describe containerruntimeconfig/<name>

# View generated MachineConfig
oc get mc | grep containerruntime
oc describe mc/99-worker-generated-containerruntime

# Check CRI-O config on node
oc debug node/<node-name>
chroot /host
cat /etc/crio/crio.conf.d/01-ctrcfg-*
crictl info
```

## CRI-O Configuration Files

ContainerRuntimeConfig manages:
- `/etc/crio/crio.conf.d/01-ctrcfg-*`: Drop-in CRI-O config
- `/etc/containers/storage.conf`: Storage configuration (if overlaySize set)

**Note**: Direct edits to these files will be detected as config drift.

## Limitations

- Only CRI-O runtime supported (not Docker/containerd)
- Some advanced CRI-O options not exposed (use MachineConfig directly)
- Storage configuration changes require reboot

## Related Concepts

- **MachineConfig**: Low-level config that ContainerRuntimeConfig generates
- **KubeletConfig**: Similar high-level API for kubelet
- **MachineConfigPool**: Target for CRI-O configuration

## References

- Design doc: `docs/ContainerRuntimeConfigDesign.md`
- CRI-O configuration: https://github.com/cri-o/cri-o/blob/main/docs/crio.conf.5.md
- Rebootless updates: `docs/MachineConfigDaemon.md#rebootless-updates`

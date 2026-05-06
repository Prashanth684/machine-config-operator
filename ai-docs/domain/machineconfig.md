# MachineConfig

**API Group**: `machineconfiguration.openshift.io/v1`  
**Kind**: `MachineConfig`  
**Scope**: Cluster (non-namespaced)

## Purpose

Defines the desired OS configuration for OpenShift nodes. MachineConfigs contain Ignition configuration that specifies files, systemd units, and kernel arguments to apply to nodes. The Machine Config Controller merges multiple MachineConfigs based on pool selectors and renders a final configuration for each pool.

## Key Fields

### Spec

| Field | Type | Description |
|-------|------|-------------|
| `config` | Ignition Config | Ignition v3 format configuration (files, systemd units, users) |
| `osImageURL` | string | OS image URL for updates (legacy format) |
| `baseOSExtensionsContainerImage` | string | Extensions container image for layering |
| `kernelArguments` | []string | Kernel arguments to add to node boot |
| `extensions` | []string | Additional features to enable (e.g., usbguard, kernel modules) |
| `fips` | bool | Enable FIPS mode |

### Metadata

| Field | Purpose |
|-------|---------|
| `labels.machineconfiguration.openshift.io/role` | Target node role (master, worker, custom) |

## Lifecycle

1. **Creation**: User or operator creates MachineConfig with role label
2. **Rendering**: MCC watches MachineConfigs, merges all configs for each pool by role
3. **Pool Update**: MCC updates MachineConfigPool.status.configuration with rendered config hash
4. **Node Application**: MCD on each node detects new config, applies changes
5. **Verification**: MCD reports success/failure in Node annotations

## Rendering Process

MCO merges MachineConfigs in this order:
1. System configs (99-*)
2. User configs (sorted by name)
3. Generated configs from KubeletConfig, ContainerRuntimeConfig

**Merge behavior**:
- Files: Later configs override earlier ones (by path)
- Systemd units: Later configs override earlier ones (by name)
- Kernel args: Appended (duplicates preserved)

## Ownership Model

| Source | Name Pattern | Purpose |
|--------|--------------|---------|
| System | `99-*-generated-*` | Platform-managed configs |
| System | `00-*` | Bootstrap configs |
| User | Any other | User-defined configs |

**Rule**: Never edit system-generated MachineConfigs. Create new user configs instead.

## Examples

### Basic File Configuration

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: worker
  name: 50-custom-sysctl
spec:
  config:
    ignition:
      version: 3.2.0
    storage:
      files:
      - path: /etc/sysctl.d/99-custom.conf
        mode: 0644
        contents:
          source: data:,net.ipv4.ip_forward%3D1
```

### Systemd Unit

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: worker
  name: 50-custom-service
spec:
  config:
    ignition:
      version: 3.2.0
    systemd:
      units:
      - name: custom-service.service
        enabled: true
        contents: |
          [Unit]
          Description=Custom Service
          [Service]
          ExecStart=/usr/local/bin/custom-script.sh
          [Install]
          WantedBy=multi-user.target
```

### Kernel Arguments

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: master
  name: 50-master-kernel-args
spec:
  kernelArguments:
  - nosmt
  - audit=1
```

## Component-Specific Behavior

### Ignition Config Structure

MCO uses Ignition v3 format. Key sections:
- `storage.files`: File contents, permissions, ownership
- `systemd.units`: Systemd unit definitions
- `passwd.users`: User creation (limited support)

**Data URLs**: File contents use data URLs. Example: `data:,hello%20world` (URL-encoded)

### Config Drift Detection

MCD periodically checks if node state matches desired config:
1. Hash desired Ignition config
2. Compare with `/etc/machine-config-daemon/currentconfig`
3. If mismatch, trigger reconciliation

### Update Mechanism

When MachineConfig changes:
1. MCC renders new config
2. MachineConfigPool coordinates updates (respects maxUnavailable)
3. MCD applies config (may require reboot)
4. Node annotated with `machineconfiguration.openshift.io/currentConfig`

**Reboot behavior**: MCD reboots node if:
- Kernel arguments changed
- FIPS mode changed
- OS image changed
- Critical systemd units changed

## References

**Tier 1 Patterns**: [Controller Runtime](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/controllers.md) | [Status Conditions](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/status.md)

**Related CRDs**: [MachineConfigPool](machineconfigpool.md) | [KubeletConfig](kubeletconfig.md) | [ContainerRuntimeConfig](containerruntimeconfig.md)

**Architecture**: [Components](../architecture/components.md)

**Decisions**: [ADR-0002 Ignition Format](../decisions/adr-0002-ignition-format.md)

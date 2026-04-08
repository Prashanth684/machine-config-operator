# ADR-0001: Use rpm-ostree for Operating System Updates

**Status**: Accepted  
**Date**: 2018-Q2 (Initial MCO design)  
**Deciders**: OpenShift MCO team, CoreOS team  
**Related**: OSTree upstream, Red Hat CoreOS

## Context

OpenShift 4 needed a way to manage operating system updates across the cluster. The OS must be:
- Atomically upgradable (all-or-nothing)
- Rollbackable if updates fail
- Immutable (prevent configuration drift)
- Efficient (minimize download size)
- Consistent (all nodes run identical OS)

Traditional package managers (yum/dnf) provide package-level updates but lack atomic upgrade/rollback capabilities and don't prevent local modifications.

## Decision

Use **rpm-ostree** as the OS update mechanism for Red Hat CoreOS nodes.

**rpm-ostree** provides:
- **OSTree commits**: Git-like OS snapshots with content-addressed storage
- **Atomic updates**: New bootloader entry + filesystem tree, activate on reboot
- **Rollback**: Previous OS deployment remains, can boot to it
- **Immutability**: `/usr` is read-only, preventing drift
- **Efficient delivery**: Container images carrying OSTree payloads
- **rpm compatibility**: Layering RPMs on top of base OS tree

## How It Works

### OS Update Flow

1. **Release image** contains `rhel-coreos` container with OSTree commit
2. **MachineConfig** specifies `OSImageURL` (container image reference)
3. **MCD** detects OS update: `rpm-ostree status` vs `OSImageURL`
4. **MCD** invokes `rpm-ostree rebase` with container image
5. **rpm-ostree** pulls image, extracts OSTree commit, creates new deployment
6. **MCD** drains node and reboots
7. **Node boots** to new OS deployment
8. **MCD validates** OS version matches expected

### OSTree Deployment Model

```
/ostree/
├── repo/                    # OSTree repository (content-addressed)
└── deploy/rhcos/
    ├── deploy/
    │   ├── abc123.0/       # Previous OS (rollback target)
    │   └── def456.0/       # Current OS (active)
    └── var/                # Shared /var across deployments
```

Each deployment is a complete filesystem tree. The bootloader selects which deployment to boot.

### Container-Wrapped OSTree

The `OSImageURL` points to a container image like:
```
quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:abc123...
```

Inside the container:
- `/srv/repo/` contains OSTree repository
- Metadata specifies OSTree commit ID
- rpm-ostree extracts commit into local repo

**Why container-wrap?** Enables using container registries for OS distribution, mirroring, and access control.

## Alternatives Considered

### 1. Traditional Package Manager (yum/dnf)

**Pros**:
- Familiar to administrators
- Fine-grained package updates
- Mature tooling

**Cons**:
- No atomic updates (partial failure possible)
- No rollback mechanism
- Doesn't prevent configuration drift
- Large update sizes (all packages, not deltas)

**Rejected because**: Lack of atomicity and rollback critical for reliable cluster operations.

### 2. Image-Based Updates (full disk images)

**Pros**:
- Truly immutable
- Simple conceptual model
- Fast rollback (just change boot entry)

**Cons**:
- Massive download sizes (entire OS image)
- No layering or customization
- Bootimage update logistics complex

**Rejected because**: Update sizes prohibitive, especially for airgapped environments. OSTree provides similar benefits with efficient storage.

### 3. Container-Based OS (e.g., Flatcar Container Linux)

**Pros**:
- Immutable OS
- Atomic updates
- Container-native approach

**Cons**:
- RHEL compatibility required for support
- Need RPM layering for extensions
- Less mature in 2018 timeframe

**Rejected because**: Need RHEL-compatible OS with RPM ecosystem for enterprise requirements.

## Consequences

### Positive

- **Atomic updates**: Nodes either fully update or don't, preventing partial states
- **Rollback capability**: Failed updates can revert to previous deployment
- **Immutability**: Read-only `/usr` prevents configuration drift
- **Efficient storage**: Content-addressed deduplication
- **Efficient network**: Delta updates between OSTree commits
- **RHEL compatibility**: rpm-ostree preserves RPM package metadata
- **Extensions support**: Can layer additional RPMs (usbguard, kerberos)

### Negative

- **Reboot required**: OS updates always require reboot (OSTree deployment activation)
- **Learning curve**: Unfamiliar to administrators used to yum/dnf
- **Debugging complexity**: Multiple OS deployments, content-addressed storage
- **Immutable constraints**: Can't install packages in `/usr` (must layer)
- **Container dependency**: OS updates tied to container image distribution

### Neutral

- **Dual system**: rpm-ostree + Ignition (MachineConfig) for configuration
- **Storage overhead**: Multiple OS deployments consume disk space (~2GB per deployment)
- **Update timing**: Updates staged during working hours, activated on reboot

## Implementation Notes

### MCD Integration

MCD wraps rpm-ostree operations:
- `rpm-ostree status --json`: Check current/pending OS
- `rpm-ostree rebase <container>`: Stage OS update
- Reboot to activate staged deployment
- `rpm-ostree rollback`: Revert if validation fails

### Extensions (Layering)

RHCOS extensions add RPMs on top of base:
```yaml
spec:
  extensions:
  - usbguard
```

rpm-ostree creates layered commit with additional packages. Also requires reboot.

### First Boot Updates

`machine-config-daemon-firstboot.service`:
- Runs `Before=kubelet.service`
- Applies OS update if bootimage differs from target
- Reboots before kubelet starts
- Ensures nodes join cluster with correct OS

**Why?** Allows using older bootimages while cluster runs newer OS (reduces bootimage churn).

## Related Decisions

- **Ignition for configuration** (ADR-0002): rpm-ostree handles OS, Ignition handles config
- **Config drift detection** (ADR-0003): Immutable `/usr` enables drift detection on writable paths

## References

- rpm-ostree: https://github.com/coreos/rpm-ostree
- OSTree: https://github.com/ostreedev/ostree
- Design doc: `docs/OSUpgrades.md`
- RHCOS: https://docs.openshift.com/container-platform/latest/architecture/architecture-rhcos.html

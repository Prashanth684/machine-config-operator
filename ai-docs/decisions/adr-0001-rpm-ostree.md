---
title: Use rpm-ostree for OS Updates
status: Accepted
date: 2018-06-01
affected_components:
  - machine-config-daemon
  - machine-config-controller
---

# ADR 0001: Use rpm-ostree for OS Updates

## Status

**Accepted**

## Context

OpenShift 4 requires a mechanism to perform operating system updates on nodes in a declarative, atomic, and rollback-capable manner. Nodes run Red Hat CoreOS (RHCOS), which needs to support:
- Atomic OS updates (all-or-nothing)
- Rollback to previous OS version on failure
- Minimal downtime during updates
- Container-based delivery of OS updates
- Integration with Kubernetes node management

## Decision

Use **rpm-ostree** as the OS update mechanism for OpenShift nodes.

## Rationale

- ✅ **Atomic Updates**: rpm-ostree provides atomic OS updates - either the entire update succeeds or the system remains on the previous version
- ✅ **Rollback Support**: Built-in support for rolling back to previous OS deployments if updates fail
- ✅ **Container Integration**: Supports pulling OS updates from container registries (aligns with OpenShift's container-based architecture)
- ✅ **Immutable Infrastructure**: OSTree's immutable `/usr` aligns with immutable node philosophy
- ✅ **Production Proven**: Mature technology from Fedora CoreOS/Atomic Host projects
- ✅ **Hybrid Approach**: Combines RPM package metadata with OSTree atomic delivery

## Alternatives Considered

### Alternative 1: Traditional RPM (yum/dnf)
- **Pro**: Familiar tooling, standard RHEL approach
- **Pro**: Fine-grained package updates
- **Con**: No atomic updates - partial update failures leave system in inconsistent state
- **Con**: No built-in rollback mechanism
- **Con**: Requires mutable `/usr` (conflicts with immutable node design)
- **Why not chosen**: Lacks atomicity and rollback, which are critical for cluster stability

### Alternative 2: Image-based Updates (replace entire disk image)
- **Pro**: Fully atomic (entire disk replaced)
- **Pro**: Simple mental model
- **Con**: Large download sizes (entire OS image vs deltas)
- **Con**: Slower updates
- **Con**: More complex node provisioning
- **Why not chosen**: Inefficient for frequent updates, poor network utilization

### Alternative 3: Custom Update Mechanism
- **Pro**: Full control over update process
- **Pro**: Could optimize for OpenShift-specific needs
- **Con**: Reinventing the wheel
- **Con**: Significant development and maintenance burden
- **Con**: Less community testing and hardening
- **Why not chosen**: rpm-ostree already provides required features with production maturity

## Consequences

**Positive**:
- Nodes can safely update OS without risk of partial update corruption
- Failed updates automatically roll back, reducing operational burden
- OS updates delivered via container registry (consistent with cluster update model)
- Unified tooling for OS and cluster updates (both via CVO/MCO)

**Negative**:
- Learning curve for operators familiar with traditional RPM
- Some traditional RPM workflows don't apply (can't install individual packages on running system)
- Must reboot to apply OS updates (rpm-ostree requirement)
- Limited ability to customize OS (immutable `/usr`)

## Affected Components

- **machine-config-daemon**: Executes `rpm-ostree` commands to apply OS updates
- **machine-config-controller**: Renders MachineConfigs with `osImageURL` pointing to rpm-ostree container
- **Red Hat CoreOS**: Uses rpm-ostree as base OS technology
- **Release Image**: Packages OS updates as container images

## Mitigation

- **Learning curve**: Document rpm-ostree workflows in MCO docs
- **Customization limits**: Provide MachineConfig for files/units in `/etc` and `/var`
- **Reboot requirement**: Coordinate reboots via MachineConfigPool (maxUnavailable) to minimize service disruption
- **Debug access**: Provide `oc debug node/` for temporary overlay mounts

## Implementation Details

### OS Update Flow

```bash
# MCD performs OS update
rpm-ostree rebase --experimental <new-os-image-url>
systemctl reboot

# After reboot, verify OS version
rpm-ostree status
```

### Container Format

OS updates packaged as OCI containers with OSTree commit inside:
```
quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:...
  └── ostree commit (RHCOS)
```

### Rollback

```bash
# If update fails, MCD can rollback
rpm-ostree rollback
systemctl reboot
```

## References

- **Upstream**: [rpm-ostree](https://github.com/coreos/rpm-ostree)
- **Upstream**: [OSTree](https://github.com/ostreedev/ostree)
- **Related**: [OSUpgrades.md](../../docs/OSUpgrades.md)
- **Related**: [MachineConfig](../domain/machineconfig.md)

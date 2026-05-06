---
title: Use Ignition for Node Configuration
status: Accepted
date: 2018-06-01
affected_components:
  - machine-config-daemon
  - machine-config-controller
  - machine-config-server
---

# ADR 0002: Use Ignition for Node Configuration

## Status

**Accepted**

## Context

OpenShift 4 requires a declarative configuration format for node provisioning and updates. The configuration system must:
- Bootstrap new nodes (first boot configuration)
- Update running nodes (day-2 configuration changes)
- Support files, systemd units, kernel arguments
- Integrate with Kubernetes CRD model (MachineConfig)
- Be verifiable and secure (checksums, HTTPS)
- Work across multiple platforms (AWS, Azure, GCP, bare metal, etc.)

## Decision

Use **CoreOS Ignition** (v3 format) as the configuration format for MachineConfigs.

## Rationale

- ✅ **Declarative**: JSON-based, machine-readable format (maps cleanly to Kubernetes CRDs)
- ✅ **Bootstrap Support**: Designed for first-boot provisioning (used by all major cloud platforms)
- ✅ **Security**: Built-in support for file checksums, HTTPS fetching, certificate validation
- ✅ **Platform Agnostic**: Works across all platforms OpenShift supports
- ✅ **Immutable Design**: Aligns with immutable infrastructure philosophy (no imperative scripts)
- ✅ **Production Proven**: Used by Fedora CoreOS, Flatcar Container Linux, RHCOS
- ✅ **Active Development**: Maintained by CoreOS team, evolving with modern needs

## Alternatives Considered

### Alternative 1: cloud-init
- **Pro**: Industry standard, widely adopted
- **Pro**: Supports many cloud platforms
- **Pro**: Flexible (supports scripts, packages, etc.)
- **Con**: Imperative (runs scripts), harder to verify correctness
- **Con**: Less secure (scripts can do anything)
- **Con**: Designed for traditional VMs, not immutable infrastructure
- **Con**: Not designed for day-2 updates
- **Why not chosen**: Imperative model conflicts with declarative Kubernetes approach

### Alternative 2: Ansible Playbooks
- **Pro**: Powerful automation, many modules
- **Pro**: Familiar to many operators
- **Con**: Requires Ansible runtime on nodes (complexity, attack surface)
- **Con**: Imperative (not declarative)
- **Con**: Not designed for first-boot provisioning
- **Con**: Slower than native tools
- **Why not chosen**: Too heavy, not suitable for bootstrap, imperative model

### Alternative 3: Custom Configuration Format
- **Pro**: Full control over format and features
- **Pro**: Could optimize for OpenShift-specific needs
- **Con**: Reinventing the wheel
- **Con**: No cross-platform support
- **Con**: Significant development and maintenance burden
- **Con**: Less community testing
- **Why not chosen**: Ignition already provides required features with broad platform support

### Alternative 4: Shell Scripts
- **Pro**: Simple, flexible
- **Pro**: No special tooling required
- **Con**: Imperative (not idempotent)
- **Con**: Hard to verify correctness
- **Con**: Security risks (arbitrary code execution)
- **Con**: No structured error handling
- **Why not chosen**: Too risky, not declarative, not idempotent

## Consequences

**Positive**:
- Unified configuration format for bootstrap and day-2 updates
- Declarative configs map cleanly to MachineConfig CRD
- Platform-agnostic (works on all clouds, bare metal, vSphere, etc.)
- Secure by design (checksums, HTTPS, no arbitrary code execution)
- MCO can validate Ignition configs before applying

**Negative**:
- Limited to what Ignition supports (no arbitrary scripts)
- JSON format can be verbose (mitigated by MachineConfig abstraction)
- Learning curve for operators unfamiliar with Ignition
- Some Ignition features not supported day-2 (disks, filesystems, users)

## Affected Components

- **machine-config-daemon**: Parses Ignition configs, applies files/units to running nodes
- **machine-config-controller**: Renders MachineConfigs as Ignition v3 JSON
- **machine-config-server**: Serves Ignition configs to new nodes during bootstrap
- **Installer**: Generates bootstrap Ignition pointing to MCS

## Mitigation

- **Limited features**: Document supported Ignition sections for day-2 updates (files, systemd units, kernel args)
- **Bootstrap-only features**: Use installer for bootstrap-time features (disks, filesystems, users)
- **Learning curve**: Provide MachineConfig examples in docs
- **JSON verbosity**: MachineConfig CRD provides YAML abstraction, MCO renders to Ignition JSON

## Supported Ignition Sections

### Bootstrap Time (installer-generated Ignition)
- ✅ Files
- ✅ Systemd units
- ✅ Disks
- ✅ Filesystems
- ✅ Users
- ✅ Groups
- ✅ Links

### Day-2 Updates (MachineConfig)
- ✅ Files
- ✅ Systemd units
- ✅ Kernel arguments
- ❌ Disks (immutable)
- ❌ Filesystems (immutable)
- ❌ Users (bootstrap-time only)
- ❌ Groups (bootstrap-time only)
- ❌ Links (use files instead)

## Implementation Details

### Ignition v3 Format

MachineConfig.spec.config contains Ignition v3 JSON:

```json
{
  "ignition": {"version": "3.2.0"},
  "storage": {
    "files": [
      {
        "path": "/etc/example.conf",
        "mode": 420,
        "contents": {"source": "data:,hello%20world"}
      }
    ]
  },
  "systemd": {
    "units": [
      {
        "name": "example.service",
        "enabled": true,
        "contents": "[Unit]\n..."
      }
    ]
  }
}
```

### Config Merging

MCO merges multiple Ignition configs:
1. Later files override earlier files (by path)
2. Later systemd units override earlier units (by name)
3. Kernel args are appended

### Bootstrap Flow

```
1. Node boots with installer-generated Ignition
2. Ignition fetches config from MCS
3. Ignition applies files, units, disks, users
4. Node joins cluster
5. MCD takes over for day-2 updates
```

## References

- **Upstream**: [CoreOS Ignition](https://github.com/coreos/ignition)
- **Spec**: [Ignition v3 Specification](https://coreos.github.io/ignition/specs/)
- **Related**: [MachineConfig](../domain/machineconfig.md)
- **Related**: [MachineConfigDaemon](../architecture/components.md#3-machine-config-daemon-mcd)

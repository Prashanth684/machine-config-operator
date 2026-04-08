# ADR-0002: Adoption of Ignition Format for Machine Configuration

**Status**: Accepted  
**Date**: 2018-Q2 (Initial MCO design)  
**Deciders**: OpenShift MCO team, CoreOS team  
**Related**: CoreOS Ignition, Cloud-init alternatives

## Context

OpenShift 4 needed a declarative configuration format for:
- Initial machine provisioning (first boot)
- Ongoing configuration updates (via MCD)
- Files, systemd units, users, networking, storage

Requirements:
- Declarative (specify desired state, not imperative steps)
- Atomic (apply all changes or none)
- Validatable (catch errors before provisioning)
- Secure (no shell scripts executed as root)
- Platform-agnostic (work across cloud, bare metal, virtual)

## Decision

Adopt **CoreOS Ignition** (v2, later v3) as the machine configuration format.

**Ignition** is a declarative JSON/YAML format specifying:
- Files and directories
- Systemd units
- Users and SSH keys
- Storage/filesystems (first boot only)
- Network configuration (first boot only)

MachineConfigs contain Ignition configs, merged by RenderController and served by MCS.

## How It Works

### First Boot (Provisioning)

1. **Node boots** with Ignition URL in userdata
2. **Ignition runs** in initramfs (before pivot to real root)
3. **Fetches config** from MCS (`/config/<pool>`)
4. **Applies config**:
   - Partitions disks (if specified)
   - Formats filesystems
   - Writes files
   - Creates users
   - Enables systemd units
5. **Pivots to real root**, systemd starts units

### Updates (MCD)

1. **MCD receives** new rendered MachineConfig (contains Ignition)
2. **Calculates diff** between current and desired Ignition
3. **Validates** diff is supportable (files/units only)
4. **Applies changes**:
   - Update files in `/etc`, `/var`
   - Update systemd units, reload daemon
   - Apply OS update via rpm-ostree
5. **Drains and reboots** (if needed)
6. **Validates** on boot that state matches config

### Supported vs Unsupported Changes

**Supported** (updatable in-place):
- Files (overwrite existing)
- Systemd units (replace, reload)
- SSH keys for `core` user

**Unsupported** (first boot only):
- Disks, RAID, filesystems
- Partitions
- Users (other than `core` SSH keys)
- Directories (created first boot, not updated)
- Network devices

Attempting unsupported changes → node Degraded state.

**Why?** These are system-level changes risky to apply on running system. Better to re-provision node.

## Example

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  name: 50-worker-custom
  labels:
    machineconfiguration.openshift.io/role: worker
spec:
  config:
    ignition:
      version: 3.2.0
    storage:
      files:
      - path: /etc/chrony.conf
        mode: 0644
        overwrite: true
        contents:
          source: data:,server%20ntp.example.com%20iburst
    systemd:
      units:
      - name: chronyd.service
        enabled: true
```

This Ignition config:
- Writes `/etc/chrony.conf` with NTP server
- Enables `chronyd.service`

## Alternatives Considered

### 1. Cloud-init

**Pros**:
- Industry standard for cloud provisioning
- Supported by all major clouds
- Familiar to administrators
- Imperative scripting model (flexibility)

**Cons**:
- Imperative (shell scripts as root - security risk)
- No validation (errors only discovered at runtime)
- Not atomic (partial execution possible)
- Inconsistent across platforms
- No built-in update mechanism

**Rejected because**: Security concerns (arbitrary shell execution), lack of atomicity, poor validation.

### 2. Ansible/Puppet/Chef

**Pros**:
- Mature configuration management
- Large ecosystem
- Supports updates

**Cons**:
- Requires agent (ongoing process, resource usage)
- Complex dependencies
- Not designed for provisioning from initramfs
- Imperative (Ansible) or complex DSL (Puppet/Chef)

**Rejected because**: Too heavyweight, not designed for initial provisioning, agent model adds complexity.

### 3. Custom JSON/YAML Format

**Pros**:
- Full control over schema
- Tailored to OpenShift needs

**Cons**:
- Need to implement tooling from scratch
- Need to integrate with provisioning systems
- No existing ecosystem
- Reinventing the wheel

**Rejected because**: Ignition already solves the problem, proven in CoreOS Container Linux, active development.

## Consequences

### Positive

- **Declarative**: Specify desired state, not steps to reach it
- **Validatable**: JSON schema validation catches errors pre-deployment
- **Atomic**: Ignition applies all changes or fails (no partial state)
- **Secure**: No arbitrary shell execution, only declarative config
- **Platform-agnostic**: Works on AWS, Azure, GCP, bare metal, vSphere
- **First boot integration**: Runs in initramfs, configures before root pivot
- **Proven**: Used in CoreOS Container Linux (now Fedora CoreOS, RHCOS)
- **Active development**: CoreOS team maintains Ignition upstream
- **Merge semantics**: Multiple Ignition configs can be merged

### Negative

- **Limited update support**: Only files/units updatable in-place
- **Learning curve**: Unfamiliar format to many administrators
- **Verbosity**: JSON/YAML can be verbose for complex configs
- **No conditionals**: Purely declarative, no if/else logic
- **Remote source limitations**: Must be fetched and embedded at render time

### Neutral

- **Ignition version evolution**: v2 → v3 required migration (happened in OCP 4.6)
- **Data URLs**: File contents often encoded as `data:,` URLs (base64 or URL-encoded)
- **Merging complexity**: RenderController must implement merge logic correctly

## Implementation Notes

### Merging MachineConfigs

RenderController merges multiple MachineConfigs:
1. Sort configs by name (lexicographic: `00-` → `99-`)
2. Merge Ignition v3 configs:
   - Files: Later configs override earlier (same path)
   - Units: Later configs override earlier (same name)
   - Arrays: Concatenate (e.g., kernel arguments)

**Merge semantics**: Defined by Ignition v3 spec, not MCO-specific.

### Remote Source Resolution

MachineConfigs can reference remote files:
```yaml
source: https://example.com/config.txt
```

**At render time**:
- RenderController fetches remote content
- Embeds content into rendered config as `data:,` URL
- Prevents runtime dependency on remote sources

**Why?** Ensures configs are static, reproducible, and don't change unexpectedly.

### Update Detection

MCD calculates diff between current and desired Ignition:
- Compare file paths, contents, modes
- Compare unit names, contents, enablement
- If changes supported → apply in-place
- If changes unsupported → degrade node

### Config Drift Detection

Ignition defines expected state. MCD monitors for drift:
- fsnotify watches Ignition-managed paths
- Detect writes to files/units
- Validate content matches current config
- Mismatch → Degraded state

See ADR-0003 for details.

## Ignition Version Evolution

**Ignition v2** (OCP 4.0 - 4.5):
- Spec 2.2.0, 2.3.0
- Structure: `{ "ignition": {...}, "storage": {...}, "systemd": {...} }`

**Ignition v3** (OCP 4.6+):
- Spec 3.0.0, 3.1.0, 3.2.0
- Structure: Same top-level, refined semantics
- Better merge semantics
- Improved validation

**Migration**: OCP 4.6 migrated all configs from v2 → v3.

## Related Decisions

- **rpm-ostree for OS updates** (ADR-0001): Ignition handles config, rpm-ostree handles OS
- **Config drift detection** (ADR-0003): Ignition defines expected state, MCD detects drift

## References

- Ignition spec: https://github.com/coreos/ignition
- Ignition v3 spec: https://coreos.github.io/ignition/specs/
- Design doc: `docs/MachineConfig.md`
- Supported changes: `docs/MachineConfigDaemon.md#supported-vs-unsupported-ignition-config-changes`

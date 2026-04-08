# ADR-0003: Proactive Config Drift Detection

**Status**: Accepted  
**Date**: 2021-Q4 (OCP 4.10)  
**Deciders**: OpenShift MCO team  
**Related**: fsnotify, Ignition validation

## Context

### Problem

In OCP 4.0-4.9, MCD only validated that on-disk state matched the MachineConfig **after rebooting**. This caused issues:

1. **Delayed detection**: Config drift (manual edits) not detected until next reboot
2. **Lost context**: Weeks/months between drift and detection, admin forgets why change was made
3. **Inconvenient degradation**: Node/pool goes Degraded at random times, blocks updates
4. **No prevention**: Manual edits succeed, causing problems later

**Example scenario**:
- Admin manually edits `/etc/kubernetes/kubelet.conf` in March
- Node doesn't reboot until June
- June update fails, node Degraded
- Admin has no idea why they made the March edit
- Cluster stuck until issue resolved

### Requirements

- **Proactive detection**: Catch drift within seconds, not weeks
- **Precise attribution**: Identify which file/unit drifted
- **Non-blocking**: Detection shouldn't prevent normal operations
- **Low overhead**: Don't impact node performance
- **Actionable**: Clear recovery path for admins

## Decision

Implement **Config Drift Monitor** in MCD using **fsnotify** to watch Ignition-managed paths and proactively detect configuration drift.

### How It Works

1. **MCD starts Config Drift Monitor** after applying MachineConfig
2. **Monitor watches** all Ignition-managed files/units via fsnotify
3. **Filesystem write detected** → Monitor wakes up
4. **Monitor validates** file/unit content and permissions vs MachineConfig
5. **Mismatch detected**:
   - Log error to console
   - Emit Kubernetes event
   - Set node state to **Degraded**
   - Stop monitoring (prevent event spam)
6. **Admin fixes** drift manually or creates forcefile
7. **MCD re-validates**, clears Degraded, restarts monitor

### Watched Paths

Monitor watches all Ignition-managed paths:
- Files: All paths in `storage.files[]`
- Units: `/etc/systemd/system/<unit>`, `/etc/systemd/system/<unit>.d/`
- Dropins: Unit override directories

**Example**:
```go
watchedPaths := []string{
    "/etc/kubernetes/kubelet.conf",
    "/etc/systemd/system/kubelet.service",
    "/etc/crio/crio.conf",
    // ... all Ignition-managed paths
}
```

### fsnotify Events

Monitor subscribes to:
- `fsnotify.Write`: File modified
- `fsnotify.Create`: File created (in watched directory)
- `fsnotify.Remove`: File removed
- `fsnotify.Chmod`: Permissions changed

**Debouncing**: Aggregate rapid events (e.g., editor write-temp-rename) before validation.

## Example Flow

```
Admin SSHs to node, runs:
  $ vi /etc/kubernetes/kubelet.conf
  # Changes maxPods: 250 → 500

  ↓ (within 1 second)

Config Drift Monitor detects write event
  ↓
Validates /etc/kubernetes/kubelet.conf
  ↓
Content doesn't match MachineConfig
  ↓
Emits event: "Configuration drift detected: /etc/kubernetes/kubelet.conf"
  ↓
Sets node annotation: state=Degraded, reason="config drift"
  ↓
Stops monitoring (prevent spam)

  ↓

Admin sees Degraded state immediately:
  $ oc get nodes
  # Node shows NotReady or Degraded

  $ oc get events | grep drift
  # Warning: Configuration drift detected...

Admin has two options:

Option 1: Fix drift manually
  $ vi /etc/kubernetes/kubelet.conf
  # Change back: maxPods: 500 → 250

  ↓

MCD preflight check passes
  ↓
Reapplies current config (no-op)
  ↓
Clears Degraded, restarts monitor

Option 2: Force reboot
  $ touch /run/machine-config-daemon-force

  ↓

MCD reboots node, reapplies config
  ↓
Drift fixed, monitor restarts
```

## Alternatives Considered

### 1. Polling-Based Validation

**Approach**: MCD periodically validates all files/units (e.g., every 5 minutes)

**Pros**:
- Simpler implementation (no fsnotify)
- Detects drift eventually

**Cons**:
- Higher overhead (periodic I/O)
- Detection delay (5 min vs 1 sec)
- Wastes CPU when no drift

**Rejected because**: fsnotify provides immediate detection with lower overhead.

### 2. Periodic Reconciliation

**Approach**: MCD automatically fixes drift by rewriting files

**Pros**:
- Self-healing
- No manual intervention

**Cons**:
- Silently overwrites admin changes (confusing)
- Can cause rapid write loops
- Masks underlying issues
- Doesn't alert admin

**Rejected because**: Important to surface drift to admin, not silently fight changes.

### 3. Read-Only Filesystem

**Approach**: Mount Ignition-managed paths read-only

**Pros**:
- Prevents drift entirely
- Simple enforcement

**Cons**:
- Some paths need writable (`/etc/kubernetes/` for dynamic certs)
- Breaks valid use cases (debugging, emergency fixes)
- Overly restrictive

**Rejected because**: Too inflexible, breaks legitimate operations.

### 4. Post-Reboot Validation Only (Status Quo)

**Approach**: Keep OCP 4.0-4.9 behavior (validate after reboot)

**Pros**:
- No code changes
- No overhead

**Cons**:
- Delayed detection (weeks/months)
- Lost context
- Inconvenient degradation timing
- Doesn't help with frequent changes

**Rejected because**: Solves none of the problems.

## Consequences

### Positive

- **Immediate detection**: Drift detected in <1 second
- **Clear attribution**: Kubernetes event identifies exact file
- **Retained context**: Admin knows immediately what changed and why
- **Prevents update failures**: Drift caught before update starts
- **Low overhead**: fsnotify more efficient than polling
- **Actionable**: Clear recovery path (fix manually or forcefile)

### Negative

- **Complexity**: fsnotify integration, event handling, debouncing
- **Edge cases**: Temporary files, editor patterns, legitimate changes
- **False positives**: Some tools write temp files (need debouncing)
- **Stop on first drift**: Only reports first drifted file (must fix to continue)

### Neutral

- **Degraded state**: Node goes Degraded on drift (expected behavior)
- **Monitoring lifecycle**: Monitor starts/stops during updates
- **Forcefile escape hatch**: Admin can force reboot to fix drift

## Implementation Details

### Monitor Lifecycle

1. **Startup**: Monitor starts after MCD finishes validating current config
2. **Update**: Monitor stops before applying new config, restarts after
3. **Degraded**: Monitor stops after detecting drift
4. **Recovery**: Monitor restarts after drift fixed

**Why stop during update?** Config will "drift" from current to new, would trigger false positives.

### Debouncing Strategy

Many editors write files via temp file + rename:
```bash
vi /etc/kubernetes/kubelet.conf
  → Writes to .kubelet.conf.swp
  → Renames to kubelet.conf
```

**Debounce**:
- Wait 500ms after write event
- Aggregate multiple events to same path
- Validate once after quiet period

**Prevents**: Spurious events from editor patterns.

### Event Content

Kubernetes event includes:
- **File path**: Which file drifted
- **Current content hash**: SHA256 of actual file
- **Expected content hash**: SHA256 from MachineConfig
- **Timestamp**: When detected

**Example**:
```
Warning  ConfigDriftDetected  1s  machine-config-daemon
Configuration drift detected on /etc/kubernetes/kubelet.conf
Expected hash: sha256:abc123...
Actual hash: sha256:def456...
```

### Permissions Validation

Monitor validates both content and permissions:
- File mode (e.g., 0644)
- Ownership (usually root:root)

**Example**: Changing file to 0777 triggers drift detection.

## Recovery Paths

### Path 1: Manual Fix (Recommended)

1. Identify drifted file from event
2. Restore file to match MachineConfig
3. MCD preflight check passes
4. MCD reapplies config (no-op)
5. Monitor restarts

**When to use**: You know what changed, can revert it.

### Path 2: Forcefile (Emergency)

1. Create `/run/machine-config-daemon-force`
2. MCD bypasses preflight check
3. MCD reapplies config, reboots
4. Drift fixed after reboot

**When to use**: Don't remember what changed, need to unblock cluster.

### Path 3: Update MachineConfig (Intentional Change)

1. Create/update MachineConfig with desired state
2. RenderController merges, generates new rendered config
3. MCD applies new config
4. Monitor restarts with new expected state

**When to use**: Manual change was intentional, should be codified.

## Related Decisions

- **Ignition format** (ADR-0002): Defines expected state that monitor validates
- **rpm-ostree updates** (ADR-0001): Immutable `/usr` complements writable path monitoring

## References

- Design doc: `docs/MachineConfigDaemon.md#config-drift-detection`
- fsnotify: https://github.com/fsnotify/fsnotify
- Implementation: `pkg/daemon/drift.go`

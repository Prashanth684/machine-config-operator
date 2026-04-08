# Machine Config Operator - Component Architecture

The Machine Config Operator consists of four main components that work together to manage operating system configuration across the cluster.

## Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│  Machine Config Operator (Deployment)                       │
│  - Manages lifecycle of sub-components                      │
│  - Creates MachineConfigPools, renders configs              │
└─────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
         ▼                    ▼                    ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│ MachineConfig    │ │ MachineConfig    │ │ MachineConfig    │
│ Controller (MCC) │ │ Server (MCS)     │ │ Daemon (MCD)     │
│ (Deployment)     │ │ (DaemonSet)      │ │ (DaemonSet)      │
└──────────────────┘ └──────────────────┘ └──────────────────┘
         │                    │                    │
         │                    │                    │
         ▼                    ▼                    ▼
  Orchestrates           Serves Ignition      Applies configs
  node updates           to new nodes         to existing nodes
```

## 1. Machine Config Operator

**Binary**: `machine-config-operator`  
**Deployment**: `openshift-machine-config-operator/machine-config-operator`  
**Purpose**: Parent operator managing sub-components

### Responsibilities

- **Lifecycle Management**: Deploys and manages MCC, MCS, MCD
- **Bootstrap**: Initial cluster configuration during install
- **ClusterOperator Status**: Reports cluster operator health
- **Template Rendering**: Provides base configuration templates

### Key Controllers

None - primarily a launcher for sub-components.

### Location

- Code: `cmd/machine-config-operator/`
- Manifests: `install/`

## 2. Machine Config Controller (MCC)

**Binary**: `machine-config-controller`  
**Deployment**: `openshift-machine-config-operator/machine-config-controller`  
**Purpose**: Orchestrate configuration updates across pools

### Responsibilities

- **Render Configurations**: Merge MachineConfigs into rendered configs
- **Coordinate Updates**: Manage rollout of configs to nodes
- **Generate Templates**: Create system-owned MachineConfigs
- **High-Level Controllers**: Process KubeletConfig, ContainerRuntimeConfig

### Sub-Controllers

#### RenderController

**Purpose**: Generate rendered MachineConfigs for each pool

**Logic**:
1. Watch MachineConfigPools and MachineConfigs
2. For each pool, find all MachineConfigs matching selector
3. Sort configs lexicographically by name
4. Merge Ignition configs (later overrides earlier)
5. Fetch and embed remote sources
6. Generate `rendered-<pool>-<hash>` MachineConfig
7. Update pool's `spec.configuration.name`

**Code**: `pkg/controller/render/`

#### UpdateController

**Purpose**: Coordinate node updates across a pool

**Logic**:
1. Watch MachineConfigPool and Nodes
2. Compare node's current config vs pool's desired config
3. Respect `maxUnavailable` constraint
4. Select next node to update
5. Annotate node with `desiredConfig`
6. Wait for MCD to complete (watch `state` annotation)
7. Repeat until all nodes updated

**Annotations Used**:
- `machineconfiguration.openshift.io/desiredConfig`: Target config name
- `machineconfiguration.openshift.io/currentConfig`: Active config name
- `machineconfiguration.openshift.io/state`: Done/Working/Degraded
- `machineconfiguration.openshift.io/reason`: Degraded reason if failed

**Code**: `pkg/controller/node/`

#### TemplateController

**Purpose**: Generate system-owned MachineConfigs from templates

**Logic**:
1. Load templates from `templates/` directory
2. Read ControllerConfig for cluster-specific values (e.g., etcd CA)
3. Render templates with substitutions
4. Create MachineConfigs (prefixed `00-`, `01-`, `99-`)
5. Maintain ownership (overwrite user edits)

**Templates**:
- `templates/master/`: Master-specific configs
- `templates/worker/`: Worker-specific configs
- `templates/common/`: Common configs

**Code**: `pkg/controller/template/`

#### KubeletConfigController

**Purpose**: Convert KubeletConfig CRs to MachineConfigs

**Logic**:
1. Watch KubeletConfig resources
2. Extract kubelet configuration
3. Generate MachineConfig writing `/etc/kubernetes/kubelet.conf`
4. Name: `99-<pool>-generated-kubelet`
5. Set owner reference to KubeletConfig

**Code**: `pkg/controller/kubelet-config/`

#### ContainerRuntimeConfigController

**Purpose**: Convert ContainerRuntimeConfig CRs to MachineConfigs

**Logic**:
1. Watch ContainerRuntimeConfig resources
2. Extract CRI-O configuration
3. Generate MachineConfig writing `/etc/crio/crio.conf.d/`
4. Name: `99-<pool>-generated-containerruntime`
5. Set owner reference to ContainerRuntimeConfig

**Code**: `pkg/controller/container-runtime-config/`

### Location

- Code: `cmd/machine-config-controller/`, `pkg/controller/`
- Design: `docs/MachineConfigController.md`

## 3. Machine Config Server (MCS)

**Binary**: `machine-config-server`  
**DaemonSet**: `openshift-machine-config-operator/machine-config-server`  
**Purpose**: Serve Ignition configs to newly provisioned nodes

### Responsibilities

- **Ignition Endpoint**: HTTP server providing Ignition configs
- **Bootstrap Support**: Serve configs during cluster install
- **Pool Discovery**: Determine which pool a node belongs to
- **Config Serving**: Provide rendered MachineConfig as Ignition

### How It Works

1. **New node boots**: Gets Ignition URL from cloud-init/PXE
2. **Node requests config**: `GET /config/<pool>`
3. **MCS determines pool**: Based on node labels/metadata
4. **MCS fetches rendered config**: For that pool
5. **MCS returns Ignition**: JSON Ignition v3 config
6. **Node applies Ignition**: Configures itself before kubelet starts

### Endpoints

- `GET /config/master`: Master node Ignition
- `GET /config/worker`: Worker node Ignition
- `GET /config/<custom-pool>`: Custom pool Ignition
- `GET /healthz`: Health check

### Bootstrap Mode

During cluster install, MCS runs in bootstrap mode:
- Serves from static files (not Kubernetes API)
- Uses bootstrap Ignition configs
- Transitions to API mode after control plane up

### Location

- Code: `cmd/machine-config-server/`, `pkg/server/`
- Design: `docs/MachineConfigServer.md`

## 4. Machine Config Daemon (MCD)

**Binary**: `machine-config-daemon`  
**DaemonSet**: `openshift-machine-config-operator/machine-config-daemon`  
**Purpose**: Apply configuration updates to individual nodes

### Responsibilities

- **Apply Configs**: Update files, systemd units, OS on node
- **Coordinate Updates**: Communicate with UpdateController via annotations
- **Config Drift Detection**: Monitor for unauthorized changes
- **OS Updates**: Apply rpm-ostree upgrades
- **Node Drain**: Drain node before reboot

### Update Flow

1. **Detect desired config**: Watch node's `desiredConfig` annotation
2. **Compare with current**: Check if update needed
3. **Validate current state**: Ensure no config drift
4. **Set state=Working**: Annotate node as updating
5. **Calculate diff**: Determine what changed
6. **Apply changes**:
   - Update files in `/etc` and `/var`
   - Update systemd units
   - Apply OS update via rpm-ostree
7. **Drain node**: Evict pods respecting PDBs
8. **Reboot** (if needed): Or reload services for rebootless updates
9. **Validate**: After reboot, verify config applied
10. **Set state=Done**: Update `currentConfig` = `desiredConfig`

### Rebootless Updates

MCD can apply some changes without rebooting:

**No drain/reboot**:
- SSH key updates
- Pull secret updates
- Kubelet CA cert rotation

**Reload CRI-O only** (no reboot):
- Container signing GPG keys
- Selected registries.conf changes

**See**: `docs/MachineConfigDaemon.md#rebootless-updates`

### Config Drift Detection

MCD monitors files/units using fsnotify:
1. Watch all Ignition-managed paths
2. Detect filesystem write events
3. Verify content matches current MachineConfig
4. If mismatch: Set node to Degraded state
5. Emit Kubernetes event with details

**Recovery**: Fix file manually or create forcefile to reboot.

### First Boot Service

On new nodes, `machine-config-daemon-firstboot.service`:
- Runs before kubelet starts
- Checks for OS update in Ignition config
- Applies OS update via rpm-ostree
- Reboots if OS updated
- Then kubelet starts

This ensures nodes boot to correct OS before joining cluster.

### Location

- Code: `cmd/machine-config-daemon/`, `pkg/daemon/`
- Design: `docs/MachineConfigDaemon.md`

## Component Interactions

### Scenario 1: User Creates MachineConfig

```
User creates MachineConfig
          ↓
RenderController detects change
          ↓
Merges configs → rendered-worker-xyz
          ↓
UpdateController detects pool's config changed
          ↓
Selects nodes to update (respects maxUnavailable)
          ↓
Annotates node with desiredConfig=rendered-worker-xyz
          ↓
MCD detects desiredConfig != currentConfig
          ↓
MCD applies config, drains, reboots
          ↓
MCD updates currentConfig=rendered-worker-xyz, state=Done
          ↓
UpdateController selects next node
          ↓
Repeat until all nodes updated
```

### Scenario 2: New Node Joins Cluster

```
Node boots with Ignition URL
          ↓
Ignition requests config from MCS
          ↓
MCS determines node → worker pool
          ↓
MCS fetches rendered-worker-xyz
          ↓
MCS returns Ignition JSON
          ↓
Ignition writes files/units to disk
          ↓
machine-config-daemon-firstboot runs
          ↓
Applies OS update if needed, reboots
          ↓
Kubelet starts, node joins cluster
          ↓
MCD runs as pod on node
          ↓
MCD annotates currentConfig=rendered-worker-xyz
```

### Scenario 3: OS Update in Release Image

```
cluster-version-operator updates release image
          ↓
TemplateController detects new OSImageURL
          ↓
Updates system MachineConfig with new OSImageURL
          ↓
RenderController merges → new rendered config
          ↓
UpdateController starts rollout
          ↓
MCD detects OS update (rpm-ostree rebase)
          ↓
MCD drains node, applies OS update, reboots
          ↓
Node boots to new OS
          ↓
MCD validates OS version matches
          ↓
Updates currentConfig, state=Done
```

## Directory Structure

```
cmd/
├── machine-config-operator/      # Parent operator
├── machine-config-controller/    # MCC binary
├── machine-config-server/        # MCS binary
└── machine-config-daemon/        # MCD binary

pkg/
├── controller/                   # MCC controllers
│   ├── render/                   # RenderController
│   ├── node/                     # UpdateController
│   ├── template/                 # TemplateController
│   ├── kubelet-config/           # KubeletConfigController
│   └── container-runtime-config/ # ContainerRuntimeConfigController
├── server/                       # MCS server
└── daemon/                       # MCD daemon

templates/                        # System MachineConfig templates
├── master/                       # Master-specific
├── worker/                       # Worker-specific
└── common/                       # Common configs
```

## References

- MCC design: `docs/MachineConfigController.md`
- MCD design: `docs/MachineConfigDaemon.md`
- MCS design: `docs/MachineConfigServer.md`
- OS updates: `docs/OSUpgrades.md`

# MCO Component Architecture

## Overview

The Machine Config Operator consists of 4 main components deployed in a single repository:
1. **machine-config-operator**: Operator deployment that manages the other 3 sub-components
2. **machine-config-controller**: Deployment that renders configs and coordinates updates
3. **machine-config-daemon**: DaemonSet that applies configs on each node
4. **machine-config-server**: DaemonSet that serves Ignition configs during bootstrap

## Repository Structure

```
machine-config-operator/
├── cmd/                                    # Binary entry points
│   ├── machine-config-operator/            # Main operator binary
│   ├── machine-config-controller/          # Controller binary
│   ├── machine-config-daemon/              # Daemon binary
│   ├── machine-config-server/              # Server binary
│   ├── machine-os-builder/                 # CoreOS layering builder
│   └── apiserver-watcher/                  # API server availability watcher
├── pkg/                                    # Shared libraries
│   ├── operator/                           # Operator logic
│   ├── controller/                         # Controllers (render, template, kubelet, etc.)
│   ├── daemon/                             # Daemon reconciliation logic
│   ├── server/                             # Ignition server logic
│   ├── apihelpers/                         # API utilities
│   ├── constants/                          # Shared constants
│   └── version/                            # Version info
├── manifests/                              # Static manifests
├── templates/                              # Rendered MachineConfig templates
└── vendor/                                 # Vendored dependencies
```

## Component Details

### 1. Machine Config Operator (MCO)

**Deployment**: `openshift-machine-config-operator/machine-config-operator`

**Purpose**: Top-level operator that manages the lifecycle of MCO sub-components

**Key Responsibilities**:
- Deploy and manage MCC, MCD, MCS deployments/daemonsets
- Sync manifests (RBAC, CRDs, static configs)
- Report ClusterOperator status
- Watch cluster infrastructure (platform, network, proxy)

**Key Controllers**:
| Controller | Purpose |
|------------|---------|
| `SyncController` | Syncs manifests to cluster |
| `StatusController` | Reports ClusterOperator status |
| `InfrastructureController` | Watches infrastructure changes |

**pkg/operator/** structure:
- `sync.go`: Manifest syncing
- `status.go`: ClusterOperator status reporting
- `operator.go`: Main operator logic

### 2. Machine Config Controller (MCC)

**Deployment**: `openshift-machine-config-operator/machine-config-controller`

**Purpose**: Renders MachineConfigs and coordinates pool updates

**Key Responsibilities**:
- Render merged MachineConfigs for each pool
- Generate configs from KubeletConfig, ContainerRuntimeConfig
- Coordinate rolling updates across pools
- Manage pool status and conditions

**Key Controllers**:
| Controller | Purpose |
|------------|---------|
| `RenderController` | Merges MachineConfigs, generates `rendered-<pool>-<hash>` |
| `TemplateController` | Generates system MachineConfigs from templates |
| `KubeletConfigController` | Generates `99-<pool>-generated-kubelet` from KubeletConfig |
| `ContainerRuntimeConfigController` | Generates `99-<pool>-generated-containerruntime` from ContainerRuntimeConfig |
| `NodeController` | Tracks node status, coordinates updates |
| `UpdateController` | Orchestrates pool updates (respects maxUnavailable) |

**pkg/controller/** structure:
- `render/`: Config rendering logic
- `template/`: Template rendering
- `kubelet/`: KubeletConfig controller
- `container-runtime/`: ContainerRuntimeConfig controller
- `node/`: Node status tracking
- `common/`: Shared controller utilities

**Data Flow**:
```
MachineConfig CRs
    ↓
RenderController → merged Ignition config → rendered-<pool>-<hash>
    ↓
UpdateController → identifies outdated nodes
    ↓
NodeController → cordons/drains nodes (respects maxUnavailable)
    ↓
MCD applies config on node
    ↓
NodeController → uncordons node
```

### 3. Machine Config Daemon (MCD)

**DaemonSet**: `openshift-machine-config-operator/machine-config-daemon`

**Purpose**: Applies configuration changes on each node

**Key Responsibilities**:
- Detect config drift (desired vs current)
- Apply Ignition configs (files, systemd units)
- Perform OS updates (rpm-ostree)
- Reboot nodes when required
- Report node state via annotations

**Reconciliation Loop**:
```
1. Read desiredConfig from node annotation
2. Compare with currentConfig
3. If different:
   a. Drain node (coordinate with MCC)
   b. Apply new config
   c. Perform OS update if needed
   d. Reboot if required
   e. Update currentConfig annotation
   f. Report state (Done, Working, Degraded)
```

**pkg/daemon/** structure:
- `daemon.go`: Main daemon logic
- `update.go`: OS update logic (rpm-ostree)
- `node.go`: Node operations (cordon, drain, reboot)
- `writer.go`: File/unit writing
- `rpm-ostree.go`: rpm-ostree operations

**Node Annotations**:
| Annotation | Purpose |
|------------|---------|
| `machineconfiguration.openshift.io/currentConfig` | Current rendered config hash |
| `machineconfiguration.openshift.io/desiredConfig` | Desired rendered config hash |
| `machineconfiguration.openshift.io/state` | Update state (Done, Working, Degraded) |
| `machineconfiguration.openshift.io/reason` | Reason for current state |

**Supported Ignition Sections**:
- ✅ Files
- ✅ Systemd units
- ✅ Kernel arguments
- ✅ Extensions
- ✅ FIPS mode
- ❌ Filesystems (bootstrap-time only)
- ❌ Disks (bootstrap-time only)
- ❌ Users (bootstrap-time only)

### 4. Machine Config Server (MCS)

**DaemonSet**: `openshift-machine-config-operator/machine-config-server`

**Purpose**: Serves Ignition configs to new nodes during bootstrap

**Key Responsibilities**:
- Serve rendered Ignition configs over HTTP/HTTPS
- Authenticate requests (bootstrap token or kubelet cert)
- Serve configs per-pool (based on node labels)
- Provide health endpoints

**pkg/server/** structure:
- `server.go`: HTTP server
- `api.go`: API handlers
- `auth.go`: Authentication

**Endpoints**:
| Endpoint | Purpose |
|----------|---------|
| `/config/<pool>` | Serve Ignition config for pool |
| `/healthz` | Health check |

**Bootstrap Flow**:
```
1. Node boots with bootstrap Ignition
2. Bootstrap Ignition points to MCS URL
3. Node requests config from MCS (authenticated with bootstrap token)
4. MCS returns rendered Ignition for node's pool
5. Ignition applies config
6. Node joins cluster
```

## Component Relationships

```
┌────────────────────────────────────────────────────────────┐
│                    machine-config-operator                  │
│  (Manages sub-components, reports ClusterOperator status)  │
└────────────────────────────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            ↓               ↓               ↓
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│      MCC        │ │      MCD        │ │      MCS        │
│  (Deployment)   │ │   (DaemonSet)   │ │   (DaemonSet)   │
│                 │ │                 │ │                 │
│ - Render configs│ │ - Apply configs │ │ - Serve Ignition│
│ - Coordinate    │ │ - OS updates    │ │ - Bootstrap new │
│   updates       │ │ - Reboot nodes  │ │   nodes         │
└─────────────────┘ └─────────────────┘ └─────────────────┘
         │                  │                     │
         ↓                  ↓                     ↓
┌─────────────────────────────────────────────────────────────┐
│                        API Server                            │
│  (MachineConfig, MachineConfigPool, KubeletConfig, etc.)    │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### Config Creation → Application

```
1. User creates MachineConfig
       ↓
2. MCC RenderController merges MachineConfigs for each pool
       ↓
3. MCC creates rendered-<pool>-<hash> MachineConfig
       ↓
4. MCC UpdateController updates MachineConfigPool.status.configuration
       ↓
5. MCC NodeController updates node.annotations.desiredConfig
       ↓
6. MCD detects desiredConfig ≠ currentConfig
       ↓
7. MCC NodeController cordons/drains node
       ↓
8. MCD applies config (files, units, OS update)
       ↓
9. MCD reboots if needed
       ↓
10. MCD updates node.annotations.currentConfig
       ↓
11. MCC NodeController uncordons node
       ↓
12. MCC UpdateController marks pool Updated when all nodes done
```

### OS Update Flow

```
1. New OS image in release payload
       ↓
2. MCC TemplateController generates new 99-*-generated-registries MachineConfig
       ↓
3. MCC renders new config with new osImageURL
       ↓
4. MCD downloads new OS image (rpm-ostree rebase)
       ↓
5. MCD reboots into new OS
       ↓
6. MCD verifies OS version matches desired
```

## Key Shared Libraries

| Package | Purpose |
|---------|---------|
| `pkg/apihelpers` | API client wrappers, informers |
| `pkg/constants` | Shared constants (paths, labels, annotations) |
| `pkg/version` | Version information |
| `pkg/helpers` | Utilities (file ops, hash, merge) |

## References

**Tier 1 Patterns**: [Controller Runtime](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/controllers.md) | [DaemonSet](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/daemonset.md)

**Domain**: [MachineConfig](../domain/machineconfig.md) | [MachineConfigPool](../domain/machineconfigpool.md)

**Development**: [Build & Dev](../MCO_DEVELOPMENT.md)

# MCO Development Guide

> **Generic development practices**: See [Tier 1 Development](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/development.md)

This guide covers MCO-specific development workflows, build instructions, and repository organization.

## Quick Start

### Prerequisites

- Go 1.22+ (check `go.mod` for exact version)
- Podman or Docker
- Access to an OpenShift cluster (for on-cluster testing)
- `oc` CLI configured with `KUBECONFIG`

### Build All Binaries

```bash
make binaries
```

Output: `_output/linux/<arch>/machine-config-*`

### Build Individual Components

```bash
make machine-config-daemon
make machine-config-controller
make machine-config-operator
make machine-config-server
```

## Repository Structure

```
machine-config-operator/
├── cmd/                              # Binary entry points
│   ├── machine-config-operator/      # Main operator
│   ├── machine-config-controller/    # Controller
│   ├── machine-config-daemon/        # DaemonSet
│   ├── machine-config-server/        # Server
│   ├── machine-os-builder/           # CoreOS layering
│   ├── apiserver-watcher/            # API availability
│   └── machine-config-tests-ext/     # Extended tests
├── pkg/                              # Shared libraries
│   ├── operator/                     # Operator logic
│   ├── controller/                   # Controllers
│   │   ├── render/                   # Config rendering
│   │   ├── template/                 # Template rendering
│   │   ├── kubelet/                  # KubeletConfig controller
│   │   ├── container-runtime/        # ContainerRuntimeConfig controller
│   │   ├── node/                     # Node management
│   │   └── common/                   # Shared utilities
│   ├── daemon/                       # Daemon reconciliation
│   ├── server/                       # Ignition server
│   ├── apihelpers/                   # API utilities
│   ├── constants/                    # Shared constants
│   └── version/                      # Version info
├── manifests/                        # Static manifests
│   ├── machineconfigcontroller/      # MCC manifests
│   ├── machineconfigdaemon/          # MCD manifests
│   └── machineconfigserver/          # MCS manifests
├── templates/                        # MachineConfig templates
├── vendor/                           # Vendored dependencies
├── test/                             # Test suites
│   ├── e2e/                          # End-to-end tests
│   ├── e2e-bootstrap/                # Bootstrap tests
│   └── helpers/                      # Test utilities
├── docs/                             # Documentation
├── ai-docs/                          # Agentic documentation
├── Makefile                          # Build system
├── Dockerfile                        # Container image build
└── go.mod                            # Go module definition
```

### cmd/ Organization

Each binary has its own `cmd/` directory with a `main.go`:

| Binary | Purpose |
|--------|---------|
| `machine-config-operator` | Top-level operator managing sub-components |
| `machine-config-controller` | Controller rendering configs, coordinating updates |
| `machine-config-daemon` | DaemonSet applying configs on nodes |
| `machine-config-server` | Server serving Ignition to new nodes |
| `machine-os-builder` | CoreOS layering builder |
| `apiserver-watcher` | Watches API server availability |
| `machine-config-tests-ext` | Extended test binary |

### pkg/ Organization

Shared libraries used by multiple binaries:

| Package | Purpose | Used By |
|---------|---------|---------|
| `pkg/operator` | Operator logic, sync, status | MCO |
| `pkg/controller` | Controllers (render, template, etc.) | MCC |
| `pkg/daemon` | Daemon reconciliation, OS updates | MCD |
| `pkg/server` | Ignition HTTP server | MCS |
| `pkg/apihelpers` | API client wrappers, informers | All |
| `pkg/constants` | Labels, annotations, paths | All |
| `pkg/version` | Version information | All |

## Development Workflow

### 1. Local Development (Build Binaries)

Build and run binaries locally:

```bash
# Build daemon
make machine-config-daemon

# Run locally (reads KUBECONFIG)
./_output/linux/amd64/machine-config-daemon start --v=4
```

**Use case**: Quick iteration on logic changes (no container rebuild)

### 2. On-Cluster Testing (Replace Pod)

Test changes in a real cluster by replacing a running pod:

```bash
# Build binary
make machine-config-daemon

# Find pod to replace
oc get pods -n openshift-machine-config-operator -l k8s-app=machine-config-daemon

# Copy binary to pod
oc cp _output/linux/amd64/machine-config-daemon \
  openshift-machine-config-operator/machine-config-daemon-xxxxx:/usr/bin/machine-config-daemon

# Delete pod to restart with new binary
oc delete pod -n openshift-machine-config-operator machine-config-daemon-xxxxx
```

**Use case**: Test changes against real cluster state (faster than image rebuild)

### 3. Container Image Testing (Full Build)

Build and deploy custom container images:

```bash
# Build image
podman build -t quay.io/<username>/machine-config-operator:dev .

# Push to registry
podman push quay.io/<username>/machine-config-operator:dev

# Update deployment to use custom image
oc set image deployment/machine-config-operator \
  -n openshift-machine-config-operator \
  machine-config-operator=quay.io/<username>/machine-config-operator:dev
```

**Use case**: Test manifest changes, multi-component changes, full integration

### 4. Debugging

**Attach debugger (delve)**:

```bash
# Install delve in container
oc debug node/<node> -- chroot /host bash
curl -L https://github.com/go-delve/delve/releases/download/v1.26.3/dlv_1.26.3_linux_amd64.tar.gz | tar -xz -C /usr/local/bin

# Run binary with delve
dlv exec /usr/bin/machine-config-daemon -- start --v=4
```

**View logs**:

```bash
# MCD logs (DaemonSet)
oc logs -n openshift-machine-config-operator -l k8s-app=machine-config-daemon

# MCC logs (Deployment)
oc logs -n openshift-machine-config-operator -l k8s-app=machine-config-controller

# MCO logs (Deployment)
oc logs -n openshift-machine-config-operator -l k8s-app=machine-config-operator
```

**SSH to node**:

```bash
oc debug node/<node>
chroot /host
journalctl -u machine-config-daemon
```

## Common Tasks

### Add New CRD

1. **Define types**: Add to `vendor/github.com/openshift/api/machineconfiguration/v1/types.go` (external repo)
2. **Add controller**: Create controller in `pkg/controller/<name>/`
3. **Wire controller**: Add to `cmd/machine-config-controller/main.go`
4. **Add manifests**: Create CRD in `manifests/`
5. **Add examples**: Create example in `examples/<name>.crd.yaml`

**Example**: See KubeletConfig controller in `pkg/controller/kubelet/`

### Add New Controller

1. **Create controller package**: `pkg/controller/<name>/`
2. **Implement controller**: Use controller-runtime patterns (see Tier 1)
3. **Add to MCC**: Wire in `cmd/machine-config-controller/main.go`
4. **Add tests**: Unit tests in `pkg/controller/<name>/<name>_test.go`

**Template**:

```go
package mycontroller

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type MyController struct {
	// dependencies
}

func (c *MyController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	// reconciliation logic
	return reconcile.Result{}, nil
}
```

### Update Dependencies

```bash
# Update go.mod
go get github.com/openshift/api@latest

# Vendor dependencies
go mod vendor

# Verify
go mod verify
```

### Build & Release Process

**CI/CD**: Uses Prow for testing, builds release images via CI

**Release image**: MCO is part of OpenShift release payload

**Branches**:
- `master`: Development branch
- `release-4.x`: Release branches

**Tagging**: Tags created by release team (not developers)

## Build Targets

| Target | Description |
|--------|-------------|
| `make binaries` | Build all binaries |
| `make machine-config-daemon` | Build MCD binary |
| `make machine-config-controller` | Build MCC binary |
| `make machine-config-operator` | Build MCO binary |
| `make machine-config-server` | Build MCS binary |
| `make test-unit` | Run unit tests |
| `make test-e2e` | Run E2E tests (requires cluster) |
| `make verify` | Run linters, gofmt, etc. |
| `make clean` | Remove build artifacts |

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `KUBECONFIG` | Cluster access | `~/.kube/config` |
| `GOARCH` | Target architecture | Auto-detected |
| `GO111MODULE` | Go modules | `on` |
| `GOPROXY` | Go proxy | `https://proxy.golang.org` |

## Component-Specific Notes

### MCD (Daemon)

- Runs as DaemonSet (one per node)
- Requires host filesystem access (`/host` mount)
- Uses `rpm-ostree` for OS updates (requires `chroot`)
- Writes to `/etc`, `/var` (mutable paths)

### MCC (Controller)

- Runs as Deployment (1-3 replicas, leader-elected)
- Renders Ignition configs (merges MachineConfigs)
- Coordinates node updates (respects maxUnavailable)

### MCS (Server)

- Runs as DaemonSet (usually only on control plane)
- Serves Ignition via HTTP/HTTPS (port 22623)
- Requires TLS certificates (managed by MCO)

## Development Tips

**Fast iteration**:
1. Use local builds (`make machine-config-daemon`)
2. Replace pod binary (`oc cp`)
3. Only rebuild container images for manifest changes

**Debugging crashes**:
1. Check pod logs (`oc logs`)
2. Check node journal (`oc debug node/<node>`, `journalctl -u machine-config-daemon`)
3. Enable verbose logging (`--v=4`)

**Testing config changes**:
1. Use custom pool (avoid breaking production nodes)
2. Start with 1 node (`maxUnavailable: 1`)
3. Monitor pool status (`oc get mcp -w`)

## References

**Tier 1 Development**: [Development Practices](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/development.md)

**Testing**: [MCO Testing](MCO_TESTING.md)

**Architecture**: [Components](architecture/components.md)

**Upstream**: [HACKING.md](../docs/HACKING.md) (detailed hacking guide)

# Machine Config Operator - Development Guide

> **Generic Development Practices**: See [Tier 1 Development Practices](https://github.com/openshift/enhancements/tree/master/agentic/practices/development) for Go standards, controller-runtime patterns, and CI/CD workflows.

This guide covers **MCO-specific** development practices.

## Quick Start

### Prerequisites

- Go 1.22+
- Access to OpenShift cluster (use [installer](https://github.com/openshift/installer))
- `KUBECONFIG` environment variable set
- Podman for building images

### Build Binaries

```bash
# Build specific component
make machine-config-daemon
make machine-config-controller
make machine-config-server

# Build all binaries
make binaries
```

**Binaries output**: `./_output/linux/amd64/`

## Repository Structure

```
cmd/                           # Binary entrypoints
├── machine-config-operator/   # Parent operator
├── machine-config-controller/ # MCC (orchestration)
├── machine-config-daemon/     # MCD (on-node agent)
└── machine-config-server/     # MCS (Ignition server)

pkg/
├── controller/                # MCC controllers
│   ├── render/                # RenderController
│   ├── node/                  # UpdateController  
│   ├── template/              # TemplateController
│   ├── kubelet-config/        # KubeletConfigController
│   └── container-runtime-config/
├── daemon/                    # MCD logic
└── server/                    # MCS HTTP server

templates/                     # System MachineConfig templates
├── master/                    # Master-specific
├── worker/                    # Worker-specific
└── common/                    # Common configs

manifests/                     # Deployment manifests
├── machineconfigcontroller/   # MCC deployment
├── machineconfigdaemon/       # MCD daemonset
└── machineconfigserver/       # MCS daemonset

test/
├── e2e/                       # End-to-end tests
└── e2e-bootstrap/             # Bootstrap tests
```

## Development Workflow

### 1. Local Development

Edit code, build binaries locally:

```bash
make machine-config-daemon
```

Run unit tests:

```bash
make test-unit

# Or specific package
go test -v ./pkg/daemon/...
```

### 2. Deploy to Cluster

Build and push custom image:

```bash
# Build image
make image-machine-config-daemon

# Tag and push to registry
podman tag localhost/machine-config-daemon:latest quay.io/youruser/machine-config-daemon:test
podman push quay.io/youruser/machine-config-daemon:test
```

Update DaemonSet to use your image:

```bash
oc edit daemonset/machine-config-daemon -n openshift-machine-config-operator
# Change image: to your custom image
```

**Faster iteration**: Use `oc debug node/<node>` to copy binary directly:

```bash
oc debug node/<node-name>
chroot /host
# Upload your binary, restart MCD
```

### 3. Run E2E Tests

```bash
make test-e2e
```

**Note**: E2E tests run against live cluster specified by `KUBECONFIG`.

## MCO-Specific Development Patterns

### Working with Templates

System MachineConfigs are generated from templates in `templates/`:

**Template structure**:
```
templates/master/00-master/
├── units/
│   └── kubelet.yaml           # Systemd unit template
└── files/
    └── kubelet.conf.yaml      # File template
```

**Template variables** (from ControllerConfig):
- `{{.EtcdCAData}}`: etcd CA certificate
- `{{.RootCAData}}`: Root CA certificate  
- `{{.PullSecretData}}`: Pull secret
- `{{.OSImageURL}}`: OS container image

**Testing template changes**:
1. Edit template in `templates/`
2. Rebuild MCC: `make machine-config-controller`
3. Deploy updated MCC
4. TemplateController regenerates MachineConfigs

### Working with Ignition

MachineConfigs contain Ignition v3 configs. Key patterns:

**Writing files**:
```go
ign3types.File{
    Node: ign3types.Node{
        Path: "/etc/custom.conf",
    },
    FileEmbedded1: ign3types.FileEmbedded1{
        Contents: ign3types.Resource{
            Source: dataurl.EncodeBytes([]byte("content")),
        },
        Mode: pointer.Int(0644),
    },
}
```

**Writing systemd units**:
```go
ign3types.Unit{
    Name:     "custom.service",
    Enabled:  pointer.Bool(true),
    Contents: pointer.String("[Service]\nExecStart=/usr/bin/custom"),
}
```

**Merging configs** (RenderController):
```go
// Use github.com/coreos/ignition/v2/config/merge
merged, err := merge.MergeConfig(baseConfig, overrideConfig)
```

### Working with rpm-ostree

MCD interacts with rpm-ostree for OS updates:

**Check OS status**:
```go
status, err := daemon.NodeUpdaterClient.GetStatus()
// Returns rpm-ostree status JSON
```

**Apply OS update**:
```go
err := daemon.NodeUpdaterClient.RunPivot(newImageURL)
// Stages new OS deployment, requires reboot
```

**Testing OS updates locally**:
```bash
# On node
rpm-ostree status
rpm-ostree rebase --experimental ostree-unverified-registry:quay.io/openshift-release-dev/...
systemctl reboot
```

## Bootstrap Development

### What is Bootstrap?

Bootstrap renders MachineConfigs **offline** (no API server) during cluster install. This must produce identical outputs to MCC controllers.

**Key files**:
- `cmd/machine-config-operator/bootstrap.go`: Bootstrap entrypoint
- `pkg/controller/bootstrap/`: Bootstrap rendering logic

### Bootstrap Tests

**Unit tests**:
```bash
go test ./pkg/controller/bootstrap/...
```

**E2E bootstrap tests**:
```bash
go test ./test/e2e-bootstrap/...
```

These tests:
1. Run MCC controllers with envtest (real etcd/API)
2. Controllers render MachineConfigs
3. Run bootstrap against same inputs
4. Compare outputs (must match!)

**Why critical?** Mismatch between bootstrap and controller → installation breaks.

### Adding Bootstrap Test Cases

1. Add manifests to `pkg/controller/bootstrap/testdata/bootstrap/`
2. Update `test/e2e-bootstrap/bootstrap_test.go`
3. Run tests: `go test ./test/e2e-bootstrap/ -v`

## Debugging Tips

### Check MCD logs on node

```bash
oc logs -n openshift-machine-config-operator daemonset/machine-config-daemon --tail=100
```

### Debug node directly

```bash
oc debug node/<node-name>
chroot /host

# Check MCD state
cat /run/machine-config-daemon.state

# Check current config
cat /etc/machine-config-daemon/currentconfig

# Check rpm-ostree status
rpm-ostree status

# Check Ignition-managed files
ls -la /etc/kubernetes/
```

### Force MCD to reapply config

```bash
oc debug node/<node-name>
chroot /host
touch /run/machine-config-daemon-force
# MCD will reboot node
```

### View rendered MachineConfig

```bash
oc get mc rendered-worker-abc123 -o yaml
```

### Check pool status

```bash
oc describe machineconfigpool/worker
```

## Common Gotchas

### 1. Bootstrap vs Controller Drift

**Problem**: Bootstrap and controller render different configs  
**Symptom**: Installation fails, nodes can't find initial config  
**Fix**: Run bootstrap E2E tests, fix drift

### 2. Template Variable Not Found

**Problem**: Template references `{{.MissingVar}}`  
**Symptom**: TemplateController logs errors  
**Fix**: Add variable to ControllerConfig

### 3. Ignition Version Mismatch

**Problem**: MachineConfig has v2 Ignition, but MCO expects v3  
**Symptom**: Rendering fails  
**Fix**: Use Ignition v3.2.0 spec

### 4. Config Drift Detection

**Problem**: Manual file edits trigger Degraded state  
**Symptom**: Node goes Degraded immediately  
**Fix**: Revert changes or update MachineConfig

### 5. Rebootless Update Not Working

**Problem**: Expected rebootless update, but node reboots  
**Symptom**: Unexpected downtime  
**Fix**: Check MCD logic in `pkg/daemon/update.go`, verify update qualifies for rebootless

## Development Resources

### MCO-Specific Docs

- **HACKING.md**: Detailed development instructions
- **docs/MachineConfigDaemon.md**: MCD design
- **docs/MachineConfigController.md**: MCC design
- **agentic/decisions/**: Architectural decision records

### Tier 1 Resources

- [Controller Patterns](https://github.com/openshift/enhancements/tree/master/agentic/patterns/operator): Reconciliation, status conditions
- [Testing Practices](https://github.com/openshift/enhancements/tree/master/agentic/practices/testing): Test pyramid, E2E framework
- [CI/CD](https://github.com/openshift/enhancements/tree/master/agentic/practices/cicd): Prow, release process

### External References

- Ignition spec: https://coreos.github.io/ignition/specs/
- rpm-ostree: https://github.com/coreos/rpm-ostree
- controller-runtime: https://github.com/kubernetes-sigs/controller-runtime

## Contributing

See [OpenShift Contributing Guide](https://github.com/openshift/enhancements/tree/master/agentic/practices/contributing) for:
- PR process
- Code review guidelines
- Commit message format
- Enhancement proposals

---

**Next Steps**: See `MCO_TESTING.md` for testing practices

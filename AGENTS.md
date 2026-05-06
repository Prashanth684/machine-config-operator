# Machine Config Operator - Agentic Documentation

**Component**: Machine Config Operator (MCO)  
**Repository**: openshift/machine-config-operator  
**Documentation Tier**: 2 (Component-specific)

> **Retrieval First**: Search ai-docs/ before reading. Use grep/find to locate relevant files.  
> **Generic Platform Patterns**: See [Tier 1 Hub](https://github.com/openshift/enhancements/tree/master/ai-docs)

## What is MCO?

Manages operating system configuration and updates for OpenShift nodes. Controls everything between the kernel and kubelet, including systemd, CRI-O, NetworkManager, and host configuration. Uses CoreOS Ignition for configuration and rpm-ostree for OS updates.

## Core Components

| Component | Type | Purpose |
|-----------|------|---------|
| **machine-config-operator** | Deployment | Orchestrates MCO sub-components, manages ClusterOperator status |
| **machine-config-controller** | Deployment | Renders MachineConfigs, coordinates pool updates, manages Ignition configs |
| **machine-config-daemon** | DaemonSet | Applies configs on nodes, performs OS updates, monitors drift |
| **machine-config-server** | DaemonSet | Serves Ignition configs during node bootstrap |

## Documentation Structure

```
ai-docs/
├── domain/                          # Component CRDs
│   ├── machineconfig.md             # Host configuration objects
│   ├── machineconfigpool.md         # Node pool management
│   ├── kubeletconfig.md             # Kubelet-specific config
│   └── containerruntimeconfig.md    # Container runtime config
├── architecture/
│   └── components.md                # Component internals, data flow
├── decisions/                       # Component ADRs
│   ├── adr-template.md
│   ├── adr-0001-rpm-ostree.md
│   ├── adr-0002-ignition-format.md
│   └── adr-0003-config-drift.md
├── exec-plans/                      # Feature implementation tracking
│   ├── active/                      # In-progress features
│   └── README.md                    # → Tier 1 guidance
├── references/
│   └── ecosystem.md                 # → Tier 1 links
├── MCO_DEVELOPMENT.md               # Build, dev workflow, repo structure
└── MCO_TESTING.md                   # Test suites, commands
```

## Quick Navigation

**CRDs**: [MachineConfig](ai-docs/domain/machineconfig.md) | [MachineConfigPool](ai-docs/domain/machineconfigpool.md) | [KubeletConfig](ai-docs/domain/kubeletconfig.md) | [ContainerRuntimeConfig](ai-docs/domain/containerruntimeconfig.md)  
**Arch**: [Components](ai-docs/architecture/components.md) | **Dev**: [Build](ai-docs/MCO_DEVELOPMENT.md) | [Test](ai-docs/MCO_TESTING.md) | **ADRs**: [Decisions](ai-docs/decisions/) | **Work**: [Exec-Plans](ai-docs/exec-plans/active/)

## Tier 1 Ecosystem Links

**Operator Patterns**: [Controller Runtime](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/controllers.md) | [Status Conditions](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/status.md) | [Webhooks](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/webhooks.md) | [Finalizers](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/finalizers.md) | [RBAC](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/rbac.md)

**Testing**: [Test Pyramid](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/testing.md) | [E2E Framework](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/e2e.md) | [Mock Strategies](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/mocking.md)

**Security**: [STRIDE Model](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/security.md) | [Secrets Management](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/secrets.md)

**Reliability**: [SLO Framework](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/slo.md) | [Degraded States](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/degraded.md)

**Fundamentals**: [Pod](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/pod.md) | [Node](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/node.md) | [DaemonSet](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/daemonset.md) | [ClusterOperator](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/clusteroperator.md)

**Cross-Repo ADRs**: [etcd](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/decisions/adr-etcd.md) | [CVO Orchestration](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/decisions/adr-cvo.md) | [Immutable Nodes](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/decisions/adr-immutable-nodes.md)

## Retrieval Tips

```bash
# Find domain concepts
grep -r "MachineConfig" ai-docs/domain/

# Find architecture docs
find ai-docs/architecture/ -name "*.md"

# Search decisions
grep -r "rpm-ostree" ai-docs/decisions/

# Check active features
ls ai-docs/exec-plans/active/
```

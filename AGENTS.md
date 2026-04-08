# Machine Config Operator - Agentic Documentation

**Component**: Machine Config Operator (MCO)  
**Repository**: openshift/machine-config-operator  
**Documentation Tier**: 2 (Component-specific)

> **Generic Platform Patterns**: See [Tier 1 Ecosystem Hub](https://github.com/openshift/enhancements/tree/master/agentic) for operator patterns, testing practices, security guidelines, and cross-repo ADRs.

## What is the Machine Config Operator?

The Machine Config Operator extends Kubernetes to manage the operating system itself. It manages updates and configuration changes to everything between the kernel and kubelet, including systemd, CRI-O, kubelet, kernel, NetworkManager, and more.

**Key Principle**: Treat the operating system as "just another Kubernetes component" that you can inspect and manage with `oc`.

## Core Components

- **MCD**: DaemonSet applying configs to nodes | **MCC**: Coordinates pool upgrades | **MCS**: Serves Ignition to new nodes

**Quick Start**: `oc describe clusteroperator/machine-config` | `oc describe machineconfigpool`

## Documentation Structure

```
agentic/
├── domain/                    # MCO-specific CRDs and concepts
│   ├── machineconfig.md
│   ├── machineconfigpool.md
│   ├── kubeletconfig.md
│   └── containerruntimeconfig.md
├── architecture/              # MCO component internals
│   └── components.md
├── decisions/                 # MCO-specific architectural decisions
│   ├── adr-0001-rpm-ostree-updates.md
│   ├── adr-0002-ignition-format.md
│   └── adr-0003-config-drift-detection.md
├── exec-plans/                # Feature planning and implementation
│   ├── active/                # Features being implemented
│   ├── completed/             # Completed features
│   ├── template.md            # Template for new exec-plans
│   └── README.md              # Exec-plan usage guide
├── references/
│   └── ecosystem.md           # Links to Tier 1 patterns
├── MCO_DEVELOPMENT.md         # MCO-specific development practices
└── MCO_TESTING.md             # MCO-specific test suites

docs/                          # Existing documentation
├── MachineConfigDaemon.md
├── MachineConfigController.md
├── MachineConfigServer.md
└── OSUpgrades.md
```

**Exec-Plans**: Use `active/` for new features, multi-PR tracking. See `exec-plans/README.md`.

**Platform Patterns (Tier 1)**: [Operator](https://github.com/openshift/enhancements/tree/master/agentic/patterns/operator) | [Testing](https://github.com/openshift/enhancements/tree/master/agentic/practices/testing) | [Security](https://github.com/openshift/enhancements/tree/master/agentic/practices/security) | [ADRs](https://github.com/openshift/enhancements/tree/master/agentic/decisions)

## Knowledge Graph

```
                         [AGENTS.md] ← Start here
                              │
              ┌───────────────┼───────────────┐
              │               │               │
         [domain/]      [architecture/]  [decisions/]
        CRD concepts    Component design  ADR history
              │               │               │
              └───────────────┼───────────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
       [MCO_DEVELOPMENT] [MCO_TESTING]  [exec-plans/]
       Dev practices    Test suites     Feature plans
              │               │               │
              └───────────────┴───────────────┘
                              │
                      [references/ecosystem]
                      Links to Tier 1
```

**AI Agent Path**: domain/ → architecture/ → decisions/ → MCO_DEVELOPMENT.md → MCO_TESTING.md

## External References

- [Product Docs](https://docs.openshift.com/container-platform/latest/post_installation_configuration/machine-configuration-tasks.html) | [Ignition](https://github.com/coreos/ignition) | [rpm-ostree](https://github.com/coreos/rpm-ostree)

---

**Tier 1 Hub**: https://github.com/openshift/enhancements/tree/master/agentic

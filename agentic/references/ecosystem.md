# Tier 1 Ecosystem References

This document links to generic OpenShift/Kubernetes patterns in the Tier 1 ecosystem hub. The MCO inherits these platform-wide patterns and practices.

## Operator Patterns

**Location**: [enhancements/agentic/patterns/operator/](https://github.com/openshift/enhancements/tree/master/agentic/patterns/operator)

- **Controller Runtime**: Reconciliation loops, event handling, client patterns
- **Status Conditions**: Available, Progressing, Degraded condition semantics
- **Leader Election**: Multi-replica operator patterns
- **Finalizers**: Resource cleanup patterns
- **Owner References**: Parent-child relationship management

**MCO Usage**:
- MachineConfigController uses standard reconciliation patterns
- MachineConfigPool status follows APD (Available/Progressing/Degraded) conditions
- Template controller uses owner references for generated MachineConfigs

## Testing Practices

**Location**: [enhancements/agentic/practices/testing/](https://github.com/openshift/enhancements/tree/master/agentic/practices/testing)

- **Test Pyramid**: Unit > Integration > E2E ratio and philosophy
- **E2E Framework**: OpenShift E2E test patterns and helpers
- **Integration Testing**: Testing with real Kubernetes API
- **Mock vs Real**: When to use mocks vs real dependencies

**MCO Usage**:
- See `MCO_TESTING.md` for MCO-specific test suites
- Unit tests in `pkg/controller/*/`
- E2E tests in `test/e2e/`

## Security Practices

**Location**: [enhancements/agentic/practices/security/](https://github.com/openshift/enhancements/tree/master/agentic/practices/security)

- **STRIDE Threat Model**: Threat modeling framework
- **RBAC Guidelines**: Role and ClusterRole design
- **Secrets Management**: Handling sensitive data
- **Admission Control**: Webhook security patterns

**MCO Usage**:
- RBAC defined in `install/0000_80_machine-config_00_rbac.yaml`
- Config drift detection prevents unauthorized changes
- SSH key rotation follows secure update patterns

## Reliability Practices

**Location**: [enhancements/agentic/practices/reliability/](https://github.com/openshift/enhancements/tree/master/agentic/practices/reliability)

- **SLO Framework**: Service Level Objectives and error budgets
- **Observability**: Metrics, logging, tracing patterns
- **Degraded State Handling**: How to handle and recover from degraded states
- **Rollback Strategies**: Safe update rollback patterns

**MCO Usage**:
- Prometheus metrics in `manifests/0000_90_machine-config_01_prometheus-rules.yaml`
- Node drain respects PodDisruptionBudgets for etcd quorum
- MachineConfigPool manages MaxUnavailable for safe rollouts

## Kubernetes Fundamentals

**Location**: [enhancements/agentic/domain/kubernetes/](https://github.com/openshift/enhancements/tree/master/agentic/domain/kubernetes)

- **Pod**: Pod lifecycle, container specs
- **Node**: Node management, taints, labels
- **DaemonSet**: DaemonSet patterns and update strategies
- **Deployment**: Deployment strategies

**MCO Usage**:
- MCD runs as DaemonSet on every node
- MCC runs as Deployment in openshift-machine-config-operator namespace
- Node annotations coordinate updates between MCC and MCD

## OpenShift Fundamentals

**Location**: [enhancements/agentic/domain/openshift/](https://github.com/openshift/enhancements/tree/master/agentic/domain/openshift)

- **Operators**: OpenShift operator lifecycle
- **ClusterOperator**: Cluster operator status reporting
- **Release Image**: OpenShift release image structure
- **Install/Upgrade**: Platform install and upgrade patterns

**MCO Usage**:
- Implements ClusterOperator status at `clusteroperator/machine-config`
- OS updates come from release image `rhel-coreos` component
- Bootstrap process integrates with openshift-installer

## Cross-Repository ADRs

**Location**: [enhancements/agentic/decisions/](https://github.com/openshift/enhancements/tree/master/agentic/decisions)

Platform-wide architectural decisions that affect multiple repos:

- **etcd Backend Decision**: Why etcd is used for Kubernetes state
- **API Conventions**: REST API design patterns
- **CRD Design Guidelines**: Custom Resource Definition best practices
- **Status Condition Standards**: Platform-wide condition semantics

**MCO-Specific ADRs**: See `agentic/decisions/` for MCO-specific decisions (rpm-ostree, Ignition format, etc.)

## CI/CD Practices

**Location**: [enhancements/agentic/practices/cicd/](https://github.com/openshift/enhancements/tree/master/agentic/practices/cicd)

- **Prow CI**: OpenShift CI configuration patterns
- **Release Process**: How components are released
- **Image Building**: Container image build patterns
- **Versioning**: Semantic versioning and compatibility

**MCO Usage**:
- CI config in `.ci-operator.yaml`
- Prow jobs test MCO changes
- MCO images are part of OpenShift release payload

## Documentation Standards

**Location**: [enhancements/agentic/practices/documentation/](https://github.com/openshift/enhancements/tree/master/agentic/practices/documentation)

- **Enhancement Process**: How to propose and document features
- **ADR Template**: Architectural Decision Record format
- **API Documentation**: Documenting CRDs and APIs

**MCO Usage**:
- Enhancements proposed via openshift/enhancements repo
- ADRs in `agentic/decisions/`
- API docs generated from Go types

---

**Note**: These links point to Tier 1 (ecosystem hub) documentation. MCO-specific patterns and decisions are documented in the `agentic/` directory of this repository.

**Last Updated**: 2026-04-08

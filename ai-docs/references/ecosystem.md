# Ecosystem References

This document provides links to **Tier 1 platform documentation** for generic patterns, practices, and fundamentals that apply across all OpenShift components.

> **Two-Tier Architecture**:  
> - **Tier 1** (platform-docs): Generic patterns, shared practices, cross-repo decisions  
> - **Tier 2** (this repo): MCO-specific CRDs, architecture, decisions

**Rule**: If another repo would need to duplicate this, it belongs in Tier 1.

## Tier 1 Hub

📚 **[OpenShift AI Platform Documentation](https://github.com/openshift/enhancements/tree/master/ai-docs)**

The Tier 1 hub contains:
- Operator patterns (controllers, status, webhooks, finalizers)
- Testing practices (test pyramid, E2E framework, mock strategies)
- Security practices (STRIDE, RBAC, secrets management)
- Reliability practices (SLO framework, observability, degraded states)
- Kubernetes fundamentals (Pod, Node, DaemonSet, Service, etc.)
- OpenShift fundamentals (ClusterOperator, release image, CVO, etc.)
- Cross-repo ADRs (etcd, CVO orchestration, immutable nodes, etc.)

## Operator Patterns

Generic controller-runtime patterns used across all operators:

| Pattern | Link | When to Use |
|---------|------|-------------|
| **Controller Runtime** | [controllers.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/controllers.md) | All controllers |
| **Status Conditions** | [status.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/status.md) | Reporting health/progress |
| **Webhooks** | [webhooks.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/webhooks.md) | Admission control, validation |
| **Finalizers** | [finalizers.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/finalizers.md) | Cleanup on deletion |
| **RBAC** | [rbac.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/rbac.md) | Permissions model |
| **Leader Election** | [leader-election.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/leader-election.md) | HA deployments |
| **Watches** | [watches.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/watches.md) | Tracking resource changes |
| **Caching** | [caching.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/caching.md) | Informers, listers |

## Testing Practices

Generic testing approaches used across all components:

| Practice | Link | When to Use |
|----------|------|-------------|
| **Test Pyramid** | [testing.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/testing.md) | Test strategy (60/30/10) |
| **E2E Framework** | [e2e.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/e2e.md) | End-to-end tests |
| **Mock Strategies** | [mocking.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/mocking.md) | Unit test mocks |
| **Integration Testing** | [integration.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/integration.md) | Testing with real dependencies |
| **CI/CD** | [cicd.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/cicd.md) | Prow, CI operators |

**Component-specific**: [MCO Testing](../MCO_TESTING.md)

## Security Practices

Generic security patterns used across all components:

| Practice | Link | When to Use |
|----------|------|-------------|
| **STRIDE Threat Model** | [security.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/security.md) | Threat modeling |
| **Secrets Management** | [secrets.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/secrets.md) | Handling credentials |
| **RBAC Guidelines** | [rbac.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/patterns/rbac.md) | Least privilege |
| **TLS/Certificates** | [tls.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/tls.md) | Certificate management |
| **Admission Control** | [admission.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/admission.md) | Validating/mutating webhooks |

## Reliability Practices

Generic reliability patterns used across all components:

| Practice | Link | When to Use |
|----------|------|-------------|
| **SLO Framework** | [slo.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/slo.md) | Defining reliability targets |
| **Observability** | [observability.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/observability.md) | Metrics, logs, traces |
| **Degraded States** | [degraded.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/degraded.md) | Handling failures gracefully |
| **Alerts** | [alerts.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/alerts.md) | Prometheus alerting |
| **Runbooks** | [runbooks.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/runbooks.md) | Operational playbooks |

## Kubernetes Fundamentals

Core Kubernetes concepts:

| Concept | Link | Description |
|---------|------|-------------|
| **Pod** | [pod.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/pod.md) | Smallest deployable unit |
| **Node** | [node.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/node.md) | Worker machines |
| **DaemonSet** | [daemonset.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/daemonset.md) | One pod per node |
| **Deployment** | [deployment.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/deployment.md) | Stateless replicas |
| **Service** | [service.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/service.md) | Stable network endpoint |
| **ConfigMap** | [configmap.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/configmap.md) | Configuration data |
| **Secret** | [secret.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/secret.md) | Sensitive data |
| **Namespace** | [namespace.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/namespace.md) | Isolation boundary |

## OpenShift Fundamentals

OpenShift-specific concepts:

| Concept | Link | Description |
|---------|------|-------------|
| **ClusterOperator** | [clusteroperator.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/clusteroperator.md) | Operator health reporting |
| **Release Image** | [release-image.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/release-image.md) | Cluster version payload |
| **CVO** | [cvo.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/cvo.md) | Cluster Version Operator |
| **OLM** | [olm.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/olm.md) | Operator Lifecycle Manager |
| **Image Registry** | [image-registry.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/image-registry.md) | Internal image registry |
| **Installer** | [installer.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/concepts/installer.md) | Cluster bootstrapping |

## Cross-Repo Architectural Decisions

Platform-level ADRs that affect multiple components:

| ADR | Link | Topic |
|-----|------|-------|
| **etcd** | [adr-etcd.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/decisions/adr-etcd.md) | Why etcd for platform state |
| **CVO Orchestration** | [adr-cvo.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/decisions/adr-cvo.md) | Update orchestration model |
| **Immutable Nodes** | [adr-immutable-nodes.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/decisions/adr-immutable-nodes.md) | Why immutable OS |
| **Release Image** | [adr-release-image.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/decisions/adr-release-image.md) | Single payload model |
| **ClusterOperator** | [adr-clusteroperator.md](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/decisions/adr-clusteroperator.md) | Operator health reporting |

**Component-specific ADRs**: [MCO Decisions](../decisions/)

## Workflows

Generic development workflows:

| Workflow | Link | Description |
|----------|------|-------------|
| **Enhancement Process** | [enhancements.md](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/enhancements.md) | Feature proposals |
| **Exec-Plans** | [exec-plans/](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/exec-plans/) | Implementation planning |
| **Release Process** | [releases.md](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/releases.md) | OpenShift releases |
| **Backports** | [backports.md](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/backports.md) | Cherry-picking fixes |

## How to Use This Document

**When you need generic pattern guidance**:
1. Search this document for the pattern/practice
2. Click through to Tier 1 docs
3. Apply pattern to MCO-specific code

**When documenting MCO**:
- ✅ Link to Tier 1 for generic patterns (controller-runtime, status conditions, etc.)
- ✅ Document MCO-specific usage in component docs
- ❌ Don't duplicate Tier 1 content in component docs

**Example**:
- ❌ BAD: Copy controller-runtime explanation to MCO docs
- ✅ GOOD: Link to Tier 1 controller-runtime, document MCO's specific controllers

## Contributing to Tier 1

Found a pattern/practice missing from Tier 1?

1. Check if it's truly generic (would 3+ repos benefit?)
2. If yes, propose addition to Tier 1
3. If no, document in component docs

**Propose Tier 1 additions**: [openshift/enhancements](https://github.com/openshift/enhancements/issues)

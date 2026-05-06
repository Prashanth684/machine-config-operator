# MCO Testing Guide

> **Generic testing practices**: See [Tier 1 Testing](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/testing.md)

This guide covers MCO-specific test organization, test commands, and testing patterns.

## Test Organization

MCO follows the test pyramid: **60% unit / 30% integration / 10% E2E**

```
        E2E (10%)           test/e2e/
       ▲                    - Real cluster tests
      ▲ ▲                   - Full stack integration
     ▲   ▲                  - Slow, expensive
    ▲     ▲
   ▲▲▲▲▲▲▲▲▲  Integration (30%)  test/e2e-bootstrap/, pkg/controller/*/
  ▲         ▲                     - envtest, controller tests
 ▲           ▲                    - With real dependencies
▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲  Unit (60%)      pkg/*/\*_test.go
                                  - Fast, isolated
                                  - Mock dependencies
```

## Test Suites

### Unit Tests (pkg/\*/\*_test.go)

**Location**: Co-located with code (`pkg/*/\*_test.go`)

**Run**:
```bash
# All unit tests
make test-unit

# Specific package
go test -v github.com/openshift/machine-config-operator/pkg/controller/render

# Disable caching
go test -count=1 github.com/openshift/machine-config-operator/pkg/...

# With coverage
go test -coverprofile=coverage.out github.com/openshift/machine-config-operator/pkg/...
go tool cover -html=coverage.out
```

**Coverage**: `make test-unit` reports coverage (target: 60%+)

### Integration Tests (test/e2e-bootstrap/)

**Location**: `test/e2e-bootstrap/`

**Purpose**: Bootstrap process validation (renders match controllers)

**Run**:
```bash
# Bootstrap integration tests
make test-e2e-bootstrap

# Requires: envtest binaries (auto-downloaded)
```

**What they test**:
- Bootstrap renders same configs as controllers
- Controller logic with real API server (envtest)
- KubeletConfig, ContainerRuntimeConfig rendering

**Key tests**:
- `TestBootstrapRender`: Bootstrap vs controller parity
- `TestKubeletConfigController`: KubeletConfig rendering
- `TestContainerRuntimeConfigController`: ContainerRuntimeConfig rendering

### E2E Tests (test/e2e/)

**Location**: `test/e2e/`

**Purpose**: Full cluster integration testing

**Run**:
```bash
# All E2E tests (requires cluster)
make test-e2e

# Specific test
go test -v ./test/e2e -run TestClusterOperatorStatusAvailable

# Extended tests (longer running)
make test-e2e-extended
```

**Prerequisites**:
- OpenShift cluster (4.x)
- `KUBECONFIG` set
- Cluster admin permissions

**Test categories**:
| Category | Tests | Description |
|----------|-------|-------------|
| **Operator health** | `TestClusterOperatorStatus*` | ClusterOperator conditions |
| **Config rendering** | `TestRenderConfig*` | MachineConfig rendering |
| **Pool updates** | `TestPoolUpdate*` | Rolling updates, maxUnavailable |
| **Kubelet config** | `TestKubeletConfig*` | KubeletConfig changes |
| **Container runtime** | `TestContainerRuntimeConfig*` | ContainerRuntimeConfig changes |
| **OS updates** | `TestOSUpdate*` | rpm-ostree updates, reboots |
| **Drift detection** | `TestConfigDrift*` | Config drift reconciliation |
| **Layering** | `TestOSLayering*` | CoreOS layering (extensions) |

## Component-Specific Test Patterns

### Controller Tests

**Pattern**: Use controller-runtime test utilities

```go
import (
	"testing"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestMyController(t *testing.T) {
	testEnv := &envtest.Environment{}
	cfg, err := testEnv.Start()
	// ... test logic
	testEnv.Stop()
}
```

**Example**: `pkg/controller/render/render_test.go`

### Daemon Tests

**Pattern**: Mock filesystem, systemd, rpm-ostree

```go
import (
	"testing"
	"github.com/openshift/machine-config-operator/pkg/daemon/mock"
)

func TestDaemonReconcile(t *testing.T) {
	mockFS := mock.NewMockFS()
	mockSystemd := mock.NewMockSystemd()
	// ... test logic
}
```

**Example**: `pkg/daemon/update_test.go`

### Ignition Config Tests

**Pattern**: Generate Ignition, validate structure

```go
import (
	"testing"
	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
)

func TestRenderIgnition(t *testing.T) {
	mc := newMachineConfig("test", ...)
	ign := renderIgnitionConfig(mc)
	// Validate Ignition structure
	assert.Equal(t, "3.2.0", ign.Ignition.Version)
}
```

**Example**: `pkg/controller/render/render_test.go`

## Test Commands

### Quick Smoke Test

```bash
# Build + unit tests (fast)
make binaries test-unit
```

### Pre-Commit Checks

```bash
# Linters + unit tests
make verify test-unit
```

### Full Test Suite

```bash
# All tests (requires cluster)
make test
```

### Debugging Tests

```bash
# Verbose output
go test -v -run TestMyTest ./pkg/controller/render

# Print logs
go test -v -run TestMyTest ./pkg/controller/render 2>&1 | tee test.log

# Stop on first failure
go test -v -failfast ./pkg/controller/render
```

## E2E Test Details

### Cluster Requirements

- OpenShift 4.x cluster
- At least 3 worker nodes (for pool update tests)
- Cluster admin access
- Internet access (for image pulls)

### Test Execution Flow

1. **Setup**: Create test MachineConfig, pool, etc.
2. **Action**: Apply config, trigger update
3. **Wait**: Poll for pool/node status
4. **Verify**: Check desired state reached
5. **Cleanup**: Delete test resources

### Example E2E Test

```go
func TestCustomMachineConfig(t *testing.T) {
	// 1. Setup
	mc := &mcfgv1.MachineConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "99-test-config"},
		Spec: mcfgv1.MachineConfigSpec{
			Config: runtime.RawExtension{Raw: ignitionConfig},
		},
	}
	
	// 2. Action
	_, err := cs.MachineconfigurationV1().MachineConfigs().Create(ctx, mc, metav1.CreateOptions{})
	require.NoError(t, err)
	
	// 3. Wait
	err = waitForPoolComplete(t, cs, "worker", mc.Name)
	require.NoError(t, err)
	
	// 4. Verify
	node := getRandomWorkerNode(t, cs)
	verifyFileContents(t, node, "/etc/test.conf", expectedContent)
	
	// 5. Cleanup
	cs.MachineconfigurationV1().MachineConfigs().Delete(ctx, mc.Name, metav1.DeleteOptions{})
}
```

## Bootstrap Tests

**Purpose**: Ensure bootstrap rendering matches controller rendering

**Why**: Bootstrap process runs offline (no API server). Controllers run online. They must produce identical MachineConfigs.

**Test flow**:
1. Start envtest (real API server + etcd)
2. Create test resources (ControllerConfig, pools, KubeletConfigs, etc.)
3. Run controllers, wait for rendered configs
4. Run bootstrap process with same inputs
5. Compare rendered config names (must match)

**If mismatch**: Installation will break (MCD won't find initial config)

**Example**:

```bash
# Run bootstrap tests
make test-e2e-bootstrap

# Output:
# Bootstrap rendered: rendered-worker-abc123
# Controller rendered: rendered-worker-abc123
# ✓ PASS (configs match)
```

## Test Coverage

**Current coverage** (as of last update):
- Unit tests: ~65%
- Integration tests: Bootstrap parity verified
- E2E tests: Core workflows covered

**Coverage targets**:
- Unit: 70%+
- Integration: All controllers
- E2E: Critical user workflows

**Generate coverage report**:

```bash
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

## Known Test Issues

### Flaky Tests

**E2E pool update tests**: May timeout if cluster under load
- **Mitigation**: Increase timeout, retry
- **Issue**: [#1234](https://github.com/openshift/machine-config-operator/issues/1234)

**Bootstrap tests**: May fail if envtest binaries not downloaded
- **Mitigation**: Run `make test-e2e-bootstrap` (auto-downloads)

### Test Environment Requirements

**E2E tests require**:
- ≥3 worker nodes (pool update tests)
- No pending updates (cluster must be stable)
- No paused pools

**Check cluster readiness**:

```bash
# Verify cluster stable
oc get clusteroperators
oc get machineconfigpool

# All pools should be UPDATED=True, UPDATING=False, DEGRADED=False
```

## Debugging Test Failures

### Unit Test Failures

1. Check error message
2. Run with `-v` for verbose output
3. Check test expectations vs actual
4. Use `t.Logf()` for debugging output

### E2E Test Failures

1. **Check pool status**: `oc get machineconfigpool`
2. **Check node status**: `oc get nodes`
3. **Check MCD logs**: `oc logs -n openshift-machine-config-operator -l k8s-app=machine-config-daemon`
4. **Check MCC logs**: `oc logs -n openshift-machine-config-operator -l k8s-app=machine-config-controller`
5. **Check must-gather**: `oc adm must-gather`

### Bootstrap Test Failures

1. Check both rendered configs (bootstrap vs controller)
2. Look for differences in:
   - File paths
   - Systemd units
   - Kernel arguments
   - Ignition version
3. Verify test data in `pkg/controller/bootstrap/testdata/`

## Must-Gather

**Collect cluster state for debugging**:

```bash
# Standard must-gather
oc adm must-gather

# MCO-specific must-gather
oc adm must-gather --image=quay.io/openshift/must-gather
```

**Must-gather includes**:
- MachineConfigs, MachineConfigPools
- Node status, annotations
- MCD, MCC, MCO logs
- Rendered configs
- rpm-ostree status

## CI/CD Integration

**Prow jobs**:
- `pull-ci-openshift-machine-config-operator-master-unit`: Unit tests
- `pull-ci-openshift-machine-config-operator-master-e2e-aws`: E2E tests (AWS)
- `pull-ci-openshift-machine-config-operator-master-verify`: Linters

**Required for merge**:
- All unit tests pass
- E2E tests pass (AWS)
- Linters pass

## Test Development Tips

**Write unit tests for**:
- Business logic (config rendering, merging)
- Error handling
- Edge cases

**Write E2E tests for**:
- User workflows (apply config, update pool)
- Failure scenarios (degraded nodes, rollback)
- Platform-specific behavior

**Avoid in E2E tests**:
- Testing unit-level logic (use unit tests)
- Testing internal implementation (test behavior)
- Long waits (use reasonable timeouts)

## References

**Tier 1 Testing**: [Testing Practices](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/testing.md)

**Tier 1 E2E**: [E2E Framework](https://github.com/openshift/enhancements/tree/master/ai-docs/platform/practices/e2e.md)

**Development**: [MCO Development](MCO_DEVELOPMENT.md)

**Architecture**: [Components](architecture/components.md)

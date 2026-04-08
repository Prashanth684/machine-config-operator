# Machine Config Operator - Testing Guide

> **Generic Testing Practices**: See [Tier 1 Testing Practices](https://github.com/openshift/enhancements/tree/master/agentic/practices/testing) for test pyramid philosophy, E2E framework patterns, and mock vs real strategies.

This guide covers **MCO-specific** test suites and testing practices.

## Test Organization

The MCO follows the standard Kubernetes testing pyramid:

```
        E2E Tests (slow, comprehensive)
              ▲
         Integration Tests
              ▲
          Unit Tests (fast, focused)
```

## Unit Tests

### Location

Unit tests live alongside the code they test:
- `pkg/controller/*/` - Controller unit tests
- `pkg/daemon/*/` - Daemon unit tests
- `pkg/server/*/` - Server unit tests

### Running Unit Tests

```bash
# All unit tests
make test-unit

# Specific package
go test -v ./pkg/daemon/...

# Disable caching
go test -count=1 ./pkg/...

# With coverage
go test -cover ./pkg/...
```

### Unit Test Patterns

#### Controller Tests

Test controller logic without real Kubernetes API:

```go
func TestRenderController(t *testing.T) {
    // Use fake clientset
    client := fake.NewSimpleClientset()
    
    // Create test objects
    pool := &mcfgv1.MachineConfigPool{...}
    config := &mcfgv1.MachineConfig{...}
    
    // Test rendering logic
    rendered, err := renderConfig(pool, []mcfgv1.MachineConfig{config})
    require.NoError(t, err)
    assert.Equal(t, expectedConfig, rendered)
}
```

#### Ignition Tests

Test Ignition config merging:

```go
func TestIgnitionMerge(t *testing.T) {
    base := createBaseIgnition()
    override := createOverrideIgnition()
    
    merged, err := merge.MergeConfig(base, override)
    require.NoError(t, err)
    
    // Verify file from override wins
    assert.Equal(t, overrideContent, merged.Storage.Files[0].Contents)
}
```

#### MCD Logic Tests

Test daemon update logic:

```go
func TestCalculateDiff(t *testing.T) {
    current := &mcfgv1.MachineConfig{...}
    desired := &mcfgv1.MachineConfig{...}
    
    diff, err := calculateDiff(current, desired)
    require.NoError(t, err)
    
    assert.True(t, diff.osUpdate)
    assert.Len(t, diff.changedFiles, 2)
}
```

### What to Unit Test

✅ **Do unit test**:
- Controller reconciliation logic
- Ignition config merging
- Diff calculation
- Template rendering
- Validation functions
- Helper functions

❌ **Don't unit test** (use integration/E2E instead):
- Actual Kubernetes API interactions
- Real OS updates (rpm-ostree)
- Filesystem operations on real nodes
- Network interactions

## Integration Tests

Integration tests use **envtest** (real etcd + API server, no nodes).

### Bootstrap Integration Tests

**Location**: `test/e2e-bootstrap/`

**Purpose**: Verify bootstrap rendering matches controller rendering

**How they work**:
1. Start envtest (real etcd/API server)
2. Create test manifests (ControllerConfig, MCPs, MCs)
3. Run MCC controllers, wait for rendered configs
4. Run bootstrap against same manifests
5. Compare outputs (must match!)

**Running**:
```bash
go test ./test/e2e-bootstrap/ -v
```

**Why critical?** Bootstrap runs during install (no API). Must produce same configs as controllers, or installation breaks.

### Adding Integration Tests

Create test in `pkg/controller/<name>/<name>_test.go`:

```go
func TestWithEnvtest(t *testing.T) {
    // Set up envtest
    env := &envtest.Environment{...}
    cfg, err := env.Start()
    defer env.Stop()
    
    // Create client
    client, err := client.New(cfg, ...)
    
    // Create test objects via API
    err = client.Create(ctx, testObject)
    
    // Run controller
    controller.Reconcile(ctx, req)
    
    // Verify via API
    err = client.Get(ctx, key, result)
    assert.Equal(t, expected, result)
}
```

## E2E Tests

E2E tests run against a **real OpenShift cluster**.

### Test Suites

MCO has multiple E2E test suites:

#### 1. Core E2E Tests

**Location**: `test/e2e-shared-tests/`  
**Splits**: `test/e2e-1of2/`, `test/e2e-2of2/` (parallelized in CI)

**Coverage**:
- MachineConfig CRUD operations
- Pool updates and rollouts
- KubeletConfig / ContainerRuntimeConfig
- Config drift detection
- Rebootless updates
- Node drain behavior

**Running**:
```bash
# Requires KUBECONFIG
make test-e2e
```

#### 2. On-Cluster Layering (OCL) Tests

**Location**: `test/e2e-ocl-shared/`  
**Splits**: `test/e2e-ocl-1of2/`, `test/e2e-ocl-2of2/`

**Coverage**:
- MachineOSBuild / MachineOSConfig
- Containerfile-based OS customization
- Layered OS updates

**Running**:
```bash
make test-e2e-ocl
```

#### 3. Internal Release Image (IRI) Tests

**Location**: `test/e2e-iri/`

**Coverage**:
- InternalReleaseImage controller
- Internal image registry integration

#### 4. Single Node Tests

**Location**: `test/e2e-single-node/`

**Coverage**:
- Single-node OpenShift (SNO) specific scenarios
- Bootstrap and upgrade on SNO

#### 5. Tech Preview Tests

**Location**: `test/e2e-techpreview-shared/`

**Coverage**:
- Features behind TechPreviewNoUpgrade feature gate
- Experimental functionality

### E2E Test Patterns

#### Test Structure

```go
func TestMachineConfigUpdate(t *testing.T) {
    // Get test framework (helpers, clients)
    cs := framework.NewClientSet("")
    
    // Create test MachineConfig
    mc := &mcfgv1.MachineConfig{
        ObjectMeta: metav1.ObjectMeta{
            Name: "test-config",
            Labels: map[string]string{
                "machineconfiguration.openshift.io/role": "worker",
            },
        },
        Spec: createTestSpec(),
    }
    _, err := cs.MachineConfigs().Create(context.TODO(), mc, metav1.CreateOptions{})
    require.NoError(t, err)
    defer cs.MachineConfigs().Delete(context.TODO(), mc.Name, metav1.DeleteOptions{})
    
    // Wait for pool to update
    err = helpers.WaitForPoolComplete(t, cs, "worker", mc.Name)
    require.NoError(t, err)
    
    // Verify on nodes
    nodes := helpers.GetNodesByRole(t, cs, "worker")
    for _, node := range nodes {
        helpers.AssertFileOnNode(t, cs, node, "/etc/test-file", expectedContent)
    }
}
```

#### Helper Functions

**Framework helpers** (`test/framework/`):
- `NewClientSet()`: Get MCO clientset
- `GetNodes()`: List cluster nodes
- `ExecOnNode()`: Run command on node via `oc debug`

**Test helpers** (`test/helpers/`):
- `WaitForPoolComplete()`: Wait for MCP update
- `WaitForConfigAndPoolComplete()`: Wait for specific config
- `AssertFileOnNode()`: Verify file content on node
- `GetMCDForNode()`: Get MCD pod for a node

### E2E Test Lifecycle

1. **Setup**: Create test resources (MCs, MCPs)
2. **Act**: Trigger update (create/update MC)
3. **Wait**: Wait for rollout (`WaitForPoolComplete`)
4. **Verify**: Check nodes updated correctly
5. **Cleanup**: Delete test resources (defer)

### Running E2E Tests

```bash
# All E2E tests (requires cluster)
make test-e2e

# Specific test
go test ./test/e2e-shared-tests/ -run TestMachineConfigUpdate -v

# With timeout (E2E tests can be slow)
go test ./test/e2e-shared-tests/ -timeout 60m -v
```

**Prerequisites**:
- `KUBECONFIG` set to cluster
- Cluster must be stable (all operators Available)

## Test Suites Summary

| Suite | Type | Location | Coverage |
|-------|------|----------|----------|
| Unit | Unit | `pkg/*_test.go` | Controller logic, helpers |
| Bootstrap | Integration | `test/e2e-bootstrap/` | Bootstrap vs controller parity |
| E2E Core | E2E | `test/e2e-shared-tests/` | MachineConfig CRUD, updates |
| E2E OCL | E2E | `test/e2e-ocl-shared/` | On-cluster layering |
| E2E IRI | E2E | `test/e2e-iri/` | Internal release images |
| E2E SNO | E2E | `test/e2e-single-node/` | Single-node scenarios |
| E2E TechPreview | E2E | `test/e2e-techpreview-shared/` | Experimental features |

## CI Integration

### Prow Jobs

MCO tests run in OpenShift CI (Prow):
- **pull-ci-openshift-machine-config-operator-master-unit**: Unit tests
- **pull-ci-openshift-machine-config-operator-master-e2e-1of2**: E2E tests (part 1)
- **pull-ci-openshift-machine-config-operator-master-e2e-2of2**: E2E tests (part 2)
- **pull-ci-openshift-machine-config-operator-master-e2e-ocl**: OCL tests

**Config**: `.ci-operator.yaml`, Prow config in openshift/release repo

### Test Parallelization

E2E tests split into multiple jobs for speed:
- `e2e-1of2`, `e2e-2of2`: Core tests split
- `e2e-ocl-1of2`, `e2e-ocl-2of2`: OCL tests split

**Why?** E2E tests are slow (30-60 min each), parallelization reduces CI time.

## Writing New Tests

### When to Add Unit Test

- Testing pure logic (no external dependencies)
- Testing helper functions
- Testing validation logic
- Fast feedback needed

### When to Add Integration Test

- Testing controller interactions with API
- Testing config rendering
- Testing with real etcd state
- Bootstrap parity verification

### When to Add E2E Test

- Testing full update flow
- Testing node-level behavior
- Testing cluster-wide rollout
- Testing reboot/drain scenarios

## Debugging Tests

### Failed E2E Test

1. **Check logs**:
   ```bash
   # MCC logs
   oc logs -n openshift-machine-config-operator deployment/machine-config-controller
   
   # MCD logs
   oc logs -n openshift-machine-config-operator daemonset/machine-config-daemon
   ```

2. **Check pool status**:
   ```bash
   oc describe mcp/worker
   ```

3. **Check node annotations**:
   ```bash
   oc get nodes -o yaml | grep machineconfiguration
   ```

4. **Run test locally**:
   ```bash
   go test ./test/e2e-shared-tests/ -run TestFailingTest -v
   ```

### Failed Bootstrap Test

1. **Check diff output**: Test shows both bootstrap and controller outputs
2. **Verify templates**: Check `templates/` for recent changes
3. **Check ControllerConfig**: Verify test data has correct variables

### Flaky Test

1. **Increase timeouts**: E2E tests sometimes need longer waits
2. **Add retries**: Use `wait.Poll()` for eventually-consistent checks
3. **Check for race conditions**: Parallel updates, timing issues
4. **Report in Jira**: File bug for investigation

## MCO-Specific Test Challenges

### 1. Reboots

**Challenge**: Tests that trigger reboots are slow  
**Solution**: Use rebootless update features when possible, parallelize tests

### 2. Config Drift

**Challenge**: Tests can trigger drift detection  
**Solution**: Clean up test MachineConfigs in defer, use unique names

### 3. Pool Pausing

**Challenge**: Tests can pause pools, affecting other tests  
**Solution**: Unpause in defer, use separate custom pools for tests

### 4. Shared Cluster State

**Challenge**: E2E tests share cluster, can interfere  
**Solution**: Use unique resource names, clean up in defer, avoid modifying default pools

## Test Coverage Goals

- **Unit tests**: >70% coverage for controller logic
- **Integration tests**: Bootstrap parity always verified
- **E2E tests**: All major features have E2E coverage

## Resources

### MCO-Specific

- `test/README.md`: Test suite overview
- `test/framework/`: Test framework code
- `test/helpers/`: Common test helpers

### Tier 1

- [Test Pyramid](https://github.com/openshift/enhancements/tree/master/agentic/practices/testing): Philosophy and ratios
- [E2E Framework](https://github.com/openshift/enhancements/tree/master/agentic/practices/testing/e2e): OpenShift E2E patterns
- [CI/CD](https://github.com/openshift/enhancements/tree/master/agentic/practices/cicd): Prow job configuration

---

**Next Steps**: See `MCO_DEVELOPMENT.md` for development workflow

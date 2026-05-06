# Exec-Plans

Execution plans bridge enhancements (design) and pull requests (implementation). They track active feature work with concrete implementation steps, file changes, and verification criteria.

## Structure

```
exec-plans/
├── active/          # In-progress features (exec-plans go here)
└── README.md        # This file
```

## Tier 1 Guidance

**Templates and detailed guidance live in Tier 1 (platform-docs)**:

📚 **[Exec-Plans Guide](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/exec-plans/README.md)**

This guide contains:
- Exec-plan template
- When to create an exec-plan
- How to structure an exec-plan
- Lifecycle management (active → extract knowledge → delete)
- Examples from other components

## Quick Start

1. **Read Tier 1 guide**: [Exec-Plans README](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/exec-plans/README.md)
2. **Copy template**: [template.md](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/exec-plans/template.md)
3. **Create exec-plan**: Save to `active/<feature-name>.md`
4. **Implement**: Use exec-plan to track progress
5. **Extract knowledge**: After completion, extract learnings to ADRs, domain docs, architecture docs
6. **Delete**: Remove exec-plan (keep ADRs, not exec-plans)

## When to Create an Exec-Plan

Create an exec-plan when:
- ✅ Feature requires changes across multiple files/packages
- ✅ Enhancement is approved, implementation details need planning
- ✅ Complex feature with multiple phases
- ✅ Team collaboration requires shared understanding of approach

Skip exec-plan when:
- ❌ Simple bug fix (< 3 files)
- ❌ Enhancement not yet approved
- ❌ Trivial feature (use PR directly)

## Example Exec-Plans

See Tier 1 for examples from other components:
- [Example 1: CVO Precaching](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/exec-plans/examples/cvo-precaching.md)
- [Example 2: Operator Lifecycle](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/exec-plans/examples/olm-bundledeployment.md)

## Workflow

```
Enhancement approved
       ↓
Create exec-plan in active/
       ↓
Implement (track progress in exec-plan)
       ↓
Complete implementation
       ↓
Extract knowledge:
  - ADRs (decisions/)
  - Domain concepts (domain/)
  - Architecture updates (architecture/)
       ↓
Delete exec-plan
```

## Why Delete After Completion?

Exec-plans are **temporary planning artifacts**. After implementation:
- **Decisions** → Extract to ADRs (permanent)
- **Architecture** → Extract to architecture docs (permanent)
- **Domain knowledge** → Extract to domain docs (permanent)
- **Exec-plan** → Delete (temporary, no longer accurate)

**Rationale**: Code evolves, exec-plans become stale. Keep decisions/architecture, not implementation plans.

## References

**Tier 1 Guide**: [Exec-Plans](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/exec-plans/README.md)

**Template**: [template.md](https://github.com/openshift/enhancements/tree/master/ai-docs/workflows/exec-plans/template.md)

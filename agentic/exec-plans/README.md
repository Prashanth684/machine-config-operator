# Execution Plans

Track active feature implementations and completed work.

## Usage

**Starting a new feature:**
```bash
cp template.md active/feature-name.md
# Fill in the template with your feature details
```

**When implementation completes:**
```bash
mv active/feature-name.md completed/
```

## Structure

- `active/` - Features currently being implemented
- `completed/` - Archived completed features
- `template.md` - Template for new exec-plans

## What to Track

Create an exec-plan when:
- Implementing a new feature from an enhancement
- Major refactoring or architectural change
- Cross-repo feature (your component's portion)
- Any multi-week engineering effort

## What NOT to Track

Don't create exec-plans for:
- Bug fixes (unless major architectural fix)
- Minor refactoring
- Documentation-only changes
- Routine maintenance

Link exec-plans from AGENTS.md so they're discoverable.

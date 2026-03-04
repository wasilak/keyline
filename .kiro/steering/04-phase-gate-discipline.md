# Phase Gate Discipline for Cerebro Rewrite

## Mandatory Phase Gates

This Cerebro rewrite project follows **strict incremental development** with mandatory phase gates. This discipline prevents the catastrophic over-engineering that can break implementations.

## Phase Gate Rules

### 1. Sequential Development
- **NEVER** start Phase N+1 until Phase N gate passes
- **NEVER** implement features from future phases
- **NEVER** add "nice to have" features not in current phase spec

### 2. Gate Requirements

Every phase gate requires:

1. ✅ **All tasks complete** - Every task in `tasks.md` marked done
2. ✅ **Backend builds** - `cargo build` completes without errors or warnings
3. ✅ **Frontend builds** - `npm run build` completes without errors or warnings
4. ✅ **Tests pass** - `cargo test` and `npm test` pass all tests
5. ✅ **Manual testing complete** - 100% of manual testing checklist passes
6. ✅ **Previous phases work** - Earlier phases still function correctly
7. ✅ **Git tag created** - Phase tagged for easy rollback

### 3. Gate Failure Protocol

If ANY gate requirement fails:

1. **STOP** - Do not proceed to next phase
2. **FIX** - Address the failing requirement
3. **RE-TEST** - Verify the fix works
4. **DOCUMENT** - Note what failed and how it was fixed
5. **RETRY GATE** - Re-run full gate checklist

### 4. No Scope Creep

During phase implementation:

- ❌ "While I'm here, let me add..."
- ❌ "This would be better if..."
- ❌ "Let me refactor this to be more flexible..."
- ✅ "Does this task spec require this? No? Then don't do it."

## Cerebro Rewrite Phase Overview

### Phase 1: Project Setup & Configuration (2-3 days)
**Gate**: Rust backend and React frontend projects initialized, configuration loading works, builds succeed

### Phase 2: Backend Core Infrastructure (3-4 days)
**Gate**: TLS management works, Elasticsearch client connects, cluster manager maintains connections

### Phase 3: Authentication & Session Management (3-4 days)
**Gate**: Local user auth works, OIDC auth works, sessions persist, RBAC enforces access

### Phase 4: Frontend Core & Dashboard (3-4 days)
**Gate**: React app loads, theme switching works, dashboard displays clusters, navigation works

### Phase 5: Cluster Management Features (1 week)
**Gate**: Index operations work, cluster settings update, shard management functional

### Phase 6: Advanced Features (1 week)
**Gate**: REST console works, snapshots/templates functional, analysis tools operational

### Phase 7: Production Readiness (3-4 days)
**Gate**: Single binary builds, Docker image works, cross-platform binaries generated, documentation complete

## Why This Matters for Cerebro Rewrite

### Building from Scratch
Unlike maintaining existing code, this is a complete rewrite:
- Start with solid foundation (config, auth, cluster management)
- Build UI on working backend API
- Add features incrementally
- Test integration at each phase
- Result: **Functional at every phase**

### This Approach for Cerebro
- Build backend infrastructure first
- Add authentication before exposing APIs
- Create frontend on stable backend
- Add features layer by layer
- Working code always

## Manual Testing Discipline

### Manual Testing is Mandatory
- Automated tests complement manual testing
- Manual testing proves it works with real Elasticsearch clusters
- User performs manual testing
- Agent provides clear test instructions

### Manual Test Checklist Format
```markdown
## Phase N Gate: Manual Testing

- [ ] Test 1: Backend starts successfully
  - Setup: Configure with test cluster
  - Command: `cargo run`
  - Expected: Server starts on port 9000, logs show "Server listening"
  
- [ ] Test 2: Frontend builds and loads
  - Setup: Build frontend assets
  - Command: `npm run build && cargo run`
  - Expected: Navigate to http://localhost:9000, see login page
  
- [ ] Test 3: Cluster connection works
  - Setup: Configure Elasticsearch cluster in config.yaml
  - Command: Navigate to dashboard after login
  - Expected: See cluster health status (green/yellow/red)
```

### Passing Manual Tests
- **ALL** checklist items must pass
- **NO** "mostly works" or "good enough"
- **NO** "I'll fix it later"
- **YES** "Every item passes completely"

## Git Tagging

### Tag Format
- Phase 1: `cerebro-rewrite-phase-1-complete`
- Phase 2: `cerebro-rewrite-phase-2-complete`
- etc.

### Why Tag
- Easy rollback if next phase breaks things
- Clear progress markers for rewrite project
- Reference points for debugging

### Tagging Command
```bash
git tag -a cerebro-rewrite-phase-1-complete -m "Phase 1: Project Setup & Configuration - GATE PASSED"
git push origin cerebro-rewrite-phase-1-complete
```

## Communication

### When Completing Phase
Agent should say:
```
Phase N implementation complete. Ready for gate testing.

Manual Testing Checklist:
- [ ] Test 1...
- [ ] Test 2...

Please run these tests and confirm all pass before we proceed to Phase N+1.
```

### When Gate Passes
User confirms, then agent:
```
Phase N gate PASSED ✅

Tagging: cerebro-rewrite-phase-N-complete
Next: Phase N+1 - [Description]

Ready to proceed?
```

### When Gate Fails
```
Phase N gate FAILED ❌

Failed test: [description]
Issue: [what went wrong]

Fixing now...
```

## Discipline Checklist

Before starting any task, ask:

- [ ] Is this task in the current phase spec?
- [ ] Have all previous phase gates passed?
- [ ] Am I adding features not in the spec?
- [ ] Am I over-engineering the solution?
- [ ] Is this the simplest approach that works?
- [ ] Will this integrate properly with existing phases?

If any answer is wrong, **STOP** and reconsider.

## Cerebro-Specific Considerations

### Integration Points
- Backend must serve embedded frontend assets
- Frontend must communicate with backend API
- Authentication must work across both layers
- Configuration must support both backend and frontend needs

### Testing Requirements
- Test with real Elasticsearch clusters (7, 8, 9)
- Test with OpenSearch clusters
- Test authentication flows (local users, OIDC)
- Test cluster operations (index management, snapshots)
- Test frontend in different browsers
- Test responsive design on different screen sizes

### Build Verification
- Backend: `cargo build`, `cargo clippy`, `cargo test`
- Frontend: `npm run build`, `npm run lint`, `npm test`
- Integration: Verify embedded assets work in single binary
- Cross-platform: Test on Linux, macOS, Windows

## Remember

**The goal is working Cerebro rewrite at every step, not impressive architecture that might work someday.**

Build incrementally. Test thoroughly. Pass gates. Repeat.

Create a modern, maintainable Elasticsearch admin tool that works reliably.
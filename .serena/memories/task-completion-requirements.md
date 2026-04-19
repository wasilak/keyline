# Task Completion Requirements (Go Backend - Keyline)

When implementing tasks from specs, you MUST follow these completion criteria:

## Build Verification (MANDATORY)

After implementing Go code changes:

1. **Format code**: `task format` or `gofmt -w -s .`
2. **Run go vet**: `task lint` - fix ALL warnings
3. **Build binary**: `task build` or `go build ./cmd/keyline`
   - Build MUST succeed with no errors
   - Address ALL warnings before marking complete
4. **Run tests**: `task test` or `go test ./...`
   - Unit tests MUST pass
   - Test with race detection: `go test -race ./...`
5. **Validate config**: Test configuration loading with new options
6. **Verify startup**: Ensure service starts successfully with new configuration

A task is NOT complete if:
- Build fails
- go vet produces warnings
- Tests fail
- Configuration doesn't load
- Service doesn't start

## Integration Points (Keyline)

Ensure new code integrates with existing architecture:

1. **Routes**: Integrate with Echo router in `cmd/keyline/main.go`
2. **Auth providers**: Integrate with auth engine (OIDC/local)
3. **Session store**: Connect to Redis or in-memory store
4. **Configuration**: Work with Viper and environment variables
5. **Logging**: Use `log/slog` with context
6. **OIDC**: Test token validation and user info extraction
7. **Credential mapping**: Verify ES header injection
8. **Transport adapters**: Test forwardAuth, standalone modes

## Git Commit Requirement

After successful verification, AUTOMATICALLY commit:

```bash
# Stage all changes
git add -A

# Commit with descriptive message
git commit -m "feat: implement X\n\nTask: [task reference]"
```

Commit message format:
- Type prefix: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`
- Brief description of implementation
- Reference to task number or name

## Verification Checklist

Before marking task complete:

- [ ] Code formatted with `task format`
- [ ] No `go vet` warnings (`task lint`)
- [ ] `go build` succeeds (`task build`)
- [ ] All tests pass (`task test`)
- [ ] Race detection clean (`go test -race`)
- [ ] Configuration loads correctly
- [ ] Service starts without errors
- [ ] Integration points verified
- [ ] Changes committed

## Why This Matters

- Ensures code integrates properly with codebase
- Catches compilation/type/config errors early
- Maintains production-ready code quality
- Prevents broken builds from being committed
- Creates clear development history

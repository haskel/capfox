# Contributing to Capfox

Thank you for your interest in contributing to Capfox! This document provides guidelines and instructions for contributing.

## Getting Started

1. **Fork the repository** and clone it locally
2. **Install Go 1.21+**
3. **Install dependencies:**
   ```bash
   make deps
   ```
4. **Build the project:**
   ```bash
   make build
   ```
5. **Run tests:**
   ```bash
   make test
   ```

## Development Workflow

### Branch Naming

- `feature/` — new features
- `fix/` — bug fixes
- `docs/` — documentation updates
- `refactor/` — code refactoring

### Making Changes

1. Create a new branch from `main`
2. Make your changes
3. Run linting and tests:
   ```bash
   make fmt vet lint test
   ```
4. Commit with clear messages
5. Push and open a Pull Request

### Commit Messages

Use clear, descriptive commit messages:

```
Add support for per-IP rate limiting

- Implement token bucket algorithm
- Add cleanup routine for stale entries
- Add configuration options
```

## Code Style

- Follow standard Go conventions
- Run `make fmt` before committing
- Run `make lint` to check for issues
- Write tests for new functionality

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test -v ./internal/monitor/...
```

## Pull Request Process

1. Update documentation if needed
2. Add tests for new features
3. Ensure all tests pass
4. Update CHANGELOG if applicable
5. Request review from maintainers

## Reporting Issues

When reporting issues, please include:

- Go version (`go version`)
- OS and architecture
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or error messages

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

## Questions?

Open an issue with the `question` label or start a discussion.

Thank you for contributing! 🦊

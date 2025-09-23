# Contributing to zyn

Thank you for your interest in contributing to zyn! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Be respectful and inclusive. We're all here to build better software together.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/yourusername/zyn.git`
3. Create a new branch: `git checkout -b feature-name`
4. Make your changes
5. Run tests: `make test`
6. Submit a pull request

## Development Setup

```bash
# Install dependencies
go mod download

# Install development tools
make install-tools

# Run tests
make test

# Run linters
make lint

# Generate coverage
make coverage
```

## Testing

- All code must have tests
- Run `make test` before submitting PRs
- Aim for >80% code coverage
- Test both success and error paths

## Code Style

- Follow standard Go conventions
- Run `make lint` to check code style
- Use meaningful variable and function names
- Add comments for complex logic

## Adding New Features

### Adding a New Synapse Type

1. Create the synapse file (e.g., `newtype.go`)
2. Implement the synapse following existing patterns
3. Add comprehensive tests (`newtype_test.go`)
4. Update documentation
5. Add to prompt consistency test

### Adding a New Provider

1. Create provider directory: `providers/newprovider/`
2. Implement the Provider interface
3. Add tests with mock HTTP server
4. Document authentication requirements
5. Add example usage

## Pull Request Process

1. Update documentation for any new features
2. Ensure all tests pass: `make test-all`
3. Ensure linting passes: `make lint`
4. Update CHANGELOG.md with your changes
5. Submit PR with clear description

## Commit Messages

Use clear, descriptive commit messages:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `test:` for test additions/changes
- `refactor:` for code refactoring

Example: `feat: add Transform synapse for text transformations`

## Questions?

Open an issue for questions or discussions about potential changes.
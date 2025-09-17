# Contributing to CloudRecon

Thank you for your interest in contributing to CloudRecon! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Guidelines](#contributing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Release Process](#release-process)

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/cloudrecon.git
   cd cloudrecon
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/cloudrecon/cloudrecon.git
   ```

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional, for using Makefile)
- Docker (optional, for containerized development)

### Setup Steps

1. **Install dependencies**:
   ```bash
   make deps
   # or
   go mod download
   ```

2. **Run tests** to ensure everything works:
   ```bash
   make test
   # or
   go test ./...
   ```

3. **Build the project**:
   ```bash
   make build
   # or
   go build -o cloudrecon ./cmd/cloudrecon
   ```

4. **Set up development environment**:
   ```bash
   make dev-setup
   ```

### IDE Setup

#### Visual Studio Code

1. Install the Go extension
2. Install recommended extensions:
   - Go
   - Go Test Explorer
   - Go Outliner
   - Go Doc

#### GoLand/IntelliJ IDEA

1. Install the Go plugin
2. Configure Go SDK to use Go 1.21+

## Contributing Guidelines

### Types of Contributions

We welcome various types of contributions:

- **Bug fixes**: Fix issues in the codebase
- **Feature additions**: Add new functionality
- **Documentation**: Improve or add documentation
- **Tests**: Add or improve test coverage
- **Performance improvements**: Optimize existing code
- **Refactoring**: Improve code structure without changing functionality

### Before You Start

1. **Check existing issues** to see if your contribution is already being worked on
2. **Create an issue** for significant changes to discuss the approach
3. **Fork the repository** and create a feature branch

### Branch Naming

Use descriptive branch names:

- `feature/description` - New features
- `fix/description` - Bug fixes
- `docs/description` - Documentation changes
- `refactor/description` - Code refactoring
- `test/description` - Test improvements
- `chore/description` - Maintenance tasks

Example:
```bash
git checkout -b feature/azure-resource-graph-integration
```

## Pull Request Process

### Before Submitting

1. **Ensure tests pass**:
   ```bash
   make test
   ```

2. **Run linting**:
   ```bash
   make lint
   ```

3. **Format code**:
   ```bash
   make fmt
   ```

4. **Update documentation** if needed

5. **Add tests** for new functionality

6. **Update CHANGELOG.md** if applicable

### Pull Request Template

When creating a pull request, please include:

- **Description**: What changes were made and why
- **Type**: Bug fix, feature, documentation, etc.
- **Testing**: How the changes were tested
- **Breaking Changes**: Any breaking changes
- **Related Issues**: Link to related issues

### Review Process

1. **Automated checks** must pass (CI/CD pipeline)
2. **Code review** by maintainers
3. **Testing** by maintainers
4. **Approval** from at least one maintainer

## Issue Reporting

### Before Creating an Issue

1. **Search existing issues** to avoid duplicates
2. **Check if it's a question** that should go to discussions
3. **Verify it's a bug** or valid feature request

### Bug Reports

Include the following information:

- **CloudRecon version**
- **Operating system**
- **Go version**
- **Steps to reproduce**
- **Expected behavior**
- **Actual behavior**
- **Error messages/logs**
- **Screenshots** (if applicable)

### Feature Requests

Include:

- **Use case**: Why is this feature needed?
- **Proposed solution**: How should it work?
- **Alternatives**: Other solutions considered
- **Additional context**: Any other relevant information

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use `golint` for linting
- Follow the project's existing patterns

### Code Organization

- **Package structure**: Follow Go conventions
- **Naming**: Use descriptive, clear names
- **Comments**: Document exported functions and types
- **Error handling**: Use proper error handling patterns

### Example Code Style

```go
// Package example demonstrates proper Go code style.
package example

import (
    "context"
    "fmt"
)

// ExampleType represents an example type with proper documentation.
type ExampleType struct {
    // Field1 is a public field with documentation.
    Field1 string
    
    // field2 is a private field.
    field2 int
}

// NewExampleType creates a new ExampleType instance.
func NewExampleType(field1 string) *ExampleType {
    return &ExampleType{
        Field1: field1,
        field2: 0,
    }
}

// DoSomething performs an operation with proper error handling.
func (e *ExampleType) DoSomething(ctx context.Context) error {
    if ctx == nil {
        return fmt.Errorf("context cannot be nil")
    }
    
    // Implementation here
    return nil
}
```

## Testing

### Test Requirements

- **Unit tests** for all new functionality
- **Integration tests** for complex features
- **Test coverage** should not decrease
- **Tests should be fast** and reliable

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run benchmarks
make benchmark
```

### Writing Tests

```go
func TestExampleFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "test",
            wantErr:  false,
        },
        {
            name:     "empty input",
            input:    "",
            expected: "",
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ExampleFunction(tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Documentation

### Documentation Types

- **Code documentation**: Comments and docstrings
- **API documentation**: Function and type documentation
- **User documentation**: README, guides, tutorials
- **Developer documentation**: Contributing guide, architecture docs

### Documentation Standards

- **Clear and concise**: Easy to understand
- **Up-to-date**: Keep documentation current
- **Examples**: Include practical examples
- **Formatting**: Use proper Markdown formatting

### Updating Documentation

1. **Update relevant files** when making changes
2. **Add new documentation** for new features
3. **Remove outdated information**
4. **Test documentation** by following the steps

## Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Steps

1. **Update version** in relevant files
2. **Update CHANGELOG.md**
3. **Create release branch**
4. **Run full test suite**
5. **Create pull request**
6. **Merge after review**
7. **Create GitHub release**
8. **Update documentation**

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and general discussion
- **Discord**: Real-time chat and support
- **Email**: support@cloudrecon.dev

### Asking Questions

When asking questions:

1. **Be specific** about your problem
2. **Include relevant information** (OS, version, etc.)
3. **Show what you've tried**
4. **Be patient** - maintainers are volunteers

## Recognition

Contributors will be recognized in:

- **CONTRIBUTORS.md** file
- **Release notes** for significant contributions
- **GitHub contributors** page
- **Project documentation**

## Thank You

Thank you for contributing to CloudRecon! Your contributions help make cloud discovery more accessible and efficient for everyone.

---

For questions about contributing, please open an issue or contact the maintainers.

# Contributing to SpecGate

Thank you for your interest in contributing to SpecGate! We welcome contributions from the community and appreciate your help in making this project better.

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed on your system:

#### Go Installation

**Linux/macOS:**
```bash
# Download and install Go 1.25+
curl -LO https://go.dev/dl/go1.25.0.linux-amd64.tar.gz  # Linux
# or
curl -LO https://go.dev/dl/go1.25.0.darwin-amd64.tar.gz  # macOS

# Extract and install
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.25.*.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin
```

**Windows:**
1. Download the installer from https://go.dev/dl/
2. Run the installer and follow the prompts
3. Verify installation: `go version`

#### Git
```bash
# Ubuntu/Debian
sudo apt-get install git

# macOS (with Homebrew)
brew install git

# Windows - download from https://git-scm.com/
```

#### Optional Tools
```bash
# UPX for binary compression (optional)
sudo apt-get install upx-ucl  # Ubuntu/Debian
brew install upx             # macOS

# Make (for using Makefile commands)
sudo apt-get install build-essential  # Ubuntu/Debian
# macOS already includes make
```

### Setting Up the Development Environment

1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/specgate.git
   cd specgate
   ```

3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/sorenjohanson/specgate.git
   ```

4. **Install dependencies**:
   ```bash
   go mod download
   ```

5. **Verify setup**:
   ```bash
   go build -o specgate
   ./specgate --help
   ```

## Development Workflow

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Keep your branch up to date**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

3. **Make your changes** following the coding standards below

4. **Test your changes**:
   ```bash
   # Build and test
   go build -o specgate
   go test ./...
   
   # Test manually with a sample OpenAPI spec
   ./specgate -spec test_specs/sample.yaml -upstream http://httpbin.org
   ```

### Testing

#### Unit Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### Manual Testing
1. Create or use an existing OpenAPI specification
2. Start SpecGate: `./specgate -spec your-spec.yaml -upstream http://your-api`
3. Make test requests through the proxy
4. Verify validation behavior and logging

### Building

```bash
# Development build
go build -o specgate

# Optimized build (production-ready)
go build -ldflags="-s -w" -o specgate

# Compressed build (requires UPX)
go build -ldflags="-s -w" -o specgate
upx --best specgate
```

## Coding Standards

### Go Style Guide
- Follow the [official Go style guide](https://go.dev/doc/effective_go)
- Use `gofmt` to format your code: `go fmt ./...`
- Use `go vet` to check for issues: `go vet ./...`

### Code Organization
- Keep functions focused and small
- Use descriptive variable and function names
- Add comments for exported functions and complex logic
- Group related functionality into separate files

### Error Handling
- Always handle errors appropriately
- Use structured logging with slog
- Provide helpful error messages for users

### Logging
- Use the existing ColoredHandler for consistent log formatting
- Log levels:
  - **ERROR**: Critical issues, validation failures
  - **WARN**: Non-critical issues, undocumented endpoints
  - **INFO**: Startup info, general status
  - **DEBUG**: Detailed debugging information

## Commit Guidelines

### Commit Message Format
```
type(scope): brief description

Detailed description if necessary

Fixes #issue_number
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```bash
feat(validation): add support for OpenAPI 3.1
fix(proxy): handle connection timeouts gracefully
docs(readme): update installation instructions
```

## Pull Request Process

1. **Ensure your changes are tested** and don't break existing functionality
2. **Update documentation** if you're adding new features
3. **Create a pull request** with:
   - Clear title and description
   - Reference to related issues
   - Screenshots/examples if applicable

4. **Respond to feedback** and make requested changes

Note commits from PRs will be squashed once merged to main.

## Common Development Tasks

### Adding a New Feature
1. Create an issue describing the feature
2. Discuss the approach in the issue
3. Create a feature branch
4. Implement the feature with tests
5. Update documentation
6. Submit a pull request

### Fixing a Bug
1. Create an issue (if one doesn't exist)
2. Write a test that reproduces the bug
3. Fix the bug
4. Ensure the test passes
5. Submit a pull request

### Improving Documentation
1. Identify areas that need improvement
2. Update the relevant documentation
3. Test any code examples
4. Submit a pull request

## Getting Help

If you need help or have questions:

1. **Check existing issues** for similar questions
2. **Open a new issue** with the "question" label
3. **Join discussions** in existing issues and PRs
4. **Read the documentation** thoroughly

## Code of Conduct

Please note that this project follows a code of conduct. By participating in this project, you agree to abide by its terms:

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Assume good intentions

## Recognition

Contributors will be recognized in the project documentation and release notes. Thank you for helping make SpecGate better!
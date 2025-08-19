# SpecGate

A lightweight OpenAPI validation proxy that validates HTTP responses against OpenAPI specifications in real-time.

[![Build](https://github.com/sorenjohanson/specgate/actions/workflows/ci.yml/badge.svg)](https://github.com/sorenjohanson/specgate/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.25-blue)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/sorenjohanson/specgate)](https://goreportcard.com/report/github.com/sorenjohanson/specgate)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Release](https://img.shields.io/github/v/release/sorenjohanson/specgate.svg)](https://github.com/sorenjohanson/specgate/releases)
[![Issues](https://img.shields.io/github/issues/sorenjohanson/specgate.svg)](https://github.com/sorenjohanson/specgate/issues)


## Features

- **Real-time validation** of HTTP responses against OpenAPI 3.x and Swagger 2.0 specifications
- **Remote spec loading** from HTTP/HTTPS URLs with safety warnings
- **Multiple validation modes**: strict, warn, report
- **Colored logging** with timestamps and structured output
- **Lightweight**: Only 3MB compressed binary
- **Zero configuration** - works out of the box

## Installation

### Download Binary

```bash
# Download the latest release
curl -L https://github.com/sorenjohanson/specgate/releases/latest/download/specgate -o specgate
chmod +x specgate
```

### Build from Source

```bash
git clone https://github.com/sorenjohanson/specgate.git
cd specgate
go build -ldflags="-s -w" -o specgate
```

## Usage

### Basic Usage

```bash
# Validate responses against a local OpenAPI spec
./specgate -spec openapi.yaml -upstream http://localhost:3000

# Use a remote OpenAPI spec
./specgate -spec https://api.example.com/openapi.json -upstream https://api.example.com

# Run in strict mode (returns 500 on validation errors)
./specgate -spec openapi.yaml -upstream http://localhost:3000 -mode strict
```

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-spec` | `openapi.yaml` | Path or URL to OpenAPI specification |
| `-upstream` | `http://localhost:3000` | Upstream API URL to proxy to |
| `-port` | `8080` | Port for the validation proxy |
| `-mode` | `warn` | Validation mode: `strict`, `warn`, or `report` |

### Validation Modes

- **`warn`** (default): Log validation errors but pass through original responses
- **`strict`**: Return HTTP 500 with error details when validation fails
- **`report`**: Log validation results for monitoring (soon!)

## How It Works

1. **Proxy Setup**: SpecGate acts as a reverse proxy between clients and your API
2. **Request Forwarding**: All requests are forwarded to your upstream API unchanged  
3. **Response Validation**: JSON responses are validated against your OpenAPI v2.0 or v3.0 spec (note: SpecGate uses [kin-openapi](https://github.com/getkin/kin-openapi) behind the scenes, 3.1 support is tracked [here](https://github.com/getkin/kin-openapi/issues/230))
4. **Logging**: Validation results are logged with colored output for easy monitoring
5. **Error Handling**: Based on the mode, invalid responses are either logged or replaced with errors

```
Client → SpecGate (validation) → Your API
       ←         (logging)     ←
```

## Examples

### Development Workflow

```bash
# Start your API server
./your-api-server &

# Run SpecGate to validate responses during development
./specgate -spec api-spec.yaml -upstream http://localhost:3000 -mode strict

# Make requests through the proxy
curl http://localhost:8080/api/users
```

### Production Monitoring

It's possible to use SpecGate in production, but it requires re-routing existing requests through the proxy, like in development. Also, please **do not** use `-mode strict` in production. This will most likely break your API and increase support overhead!

Please share any issues you find if you do end up using SpecGate in production.

```bash
./specgate \
  -spec https://api.example.com/openapi.json \
  -upstream https://api.example.com \
  -port 8080
```

### Remote Spec with Safety Check

When using a remote spec that doesn't match your upstream URL, SpecGate will warn you:

```bash
./specgate -spec https://docs.example.com/spec.json -upstream https://api.other.com
# WARNING: spec URL (https://docs.example.com) does not match upstream URL (https://api.other.com)
# Do you want to continue? (y/N):
```

## Configuration

### Logging

SpecGate provides colored, structured logging:

- 🔴 **ERROR**: Validation failures, critical issues  
- 🟡 **WARN**: Undocumented endpoints, non-critical issues
- 🟢 **INFO**: Startup information, general status

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on:

- Setting up your development environment
- Installing Go and required tools
- Development workflow and coding standards
- Testing and building procedures
- Pull request process

## License

This project is licensed under the GNU General Public License v3.0 - see [LICENSE](LICENSE) for info.

## Acknowledgments

- Built with [kin-openapi](https://github.com/getkin/kin-openapi) for OpenAPI validation
- Uses Go's standard library `log/slog` for structured logging
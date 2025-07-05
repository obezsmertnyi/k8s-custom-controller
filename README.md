# k8s-cli

A Kubernetes custom controller and CLI tool with advanced configuration management and API server capabilities.

## Features

### Configuration Management with Viper

The application uses Viper for flexible configuration management with the following priority order:

1. Command-line flags
2. Environment variables (with `KCUSTOM_` prefix)
3. Configuration file
4. Default values

If no configuration file is found, the application will use default values and environment variables, and a warning message will be logged.

### Kubernetes CLI Commands

The application includes a set of commands for managing Kubernetes deployments:

#### List Deployments

List all deployments in a namespace with detailed information.

```sh
# List deployments in default namespace
./k8s-cli list

# List deployments in specific namespace
./k8s-cli list --namespace kube-system

# Use custom kubeconfig file
./k8s-cli list --kubeconfig /path/to/kubeconfig
```

#### Create Deployment

Create a new Kubernetes deployment with configurable parameters.

```sh
# Create a basic deployment
./k8s-cli create --name my-app --image nginx:latest

# Create with custom settings
./k8s-cli create \
  --name my-app \
  --image nginx:latest \
  --replicas 3 \
  --port 80 \
  --namespace my-namespace
```

#### Delete Deployment

Delete a deployment from a namespace.

```sh
# Delete a deployment from default namespace
./k8s-cli delete my-app

# Delete from specific namespace
./k8s-cli delete my-app --namespace my-namespace
```

All Kubernetes commands support the `--kubeconfig` flag to specify a custom Kubernetes configuration file. If not provided, the default path (`~/.kube/config`) will be used.

### FastHTTP Server Command

The application includes a high-performance FastHTTP server with the following features:

- Configurable host and port settings
- Request logging with detailed metrics
- Sensible timeout defaults for production use
- 10MB maximum request size limit for security
- Graceful shutdown with signal handling

**Usage:**
```sh
# Start server on all interfaces (default)
./k8s-cli server

# Start server with custom port
./k8s-cli server --port 8090

# Start with debug logging
./k8s-cli server --log-level debug
```

### Log Level Support

The application supports different log levels using `zerolog`:

```sh
# Available log levels
./k8s-cli --log-level trace  # Most verbose
./k8s-cli --log-level debug  # Detailed debugging information
./k8s-cli --log-level info   # Default level (if not specified)
./k8s-cli --log-level warn   # Warning conditions
./k8s-cli --log-level error  # Error conditions
```

Log format is configured in the configuration file or via environment variables:

```sh
# Set log format via environment variable
export KCUSTOM_LOGGING_FORMAT=json  # For JSON format
export KCUSTOM_LOGGING_FORMAT=text  # For human-readable format (default)
```

The logging system is centralized and configured at application startup. All components respect the global logging configuration, including the log level and format settings.

## Project Structure

- `cmd/` — Contains CLI commands and configuration management
  - `root.go` — Root command and centralized logger configuration
  - `config.go` — Configuration management with Viper
  - `server.go` — FastHTTP server implementation with graceful shutdown
  - `kubernetes.go` — Kubernetes CLI commands (list, create, delete) with shared helpers
- `main.go` — Entry point for the application
- `tests/` — Test files
  - `server_test.go` — Tests for server functionality
  - `logging_test.go` — Tests for logging configuration
- `Makefile` — Build automation tasks
- `Dockerfile` — Distroless Dockerfile for secure containerization
- `.github/workflows/` — GitHub Actions workflows for CI/CD
- `charts/app` — Helm chart for Kubernetes deployment

## Development

### Building the Application

```sh
# Build the binary
make build

# Clean build artifacts
make clean

# Run linters
make lint

# Run all tests
make test

# Generate test coverage report
make coverage
```

### Docker Support

```sh
# Build Docker image
make docker-build

# Run in Docker container
make docker-run
```

### Testing Specific Components

```sh
# Test server component
make test-server

# Test logging component
make test-logging
```

## CI/CD Pipeline

The project includes a GitHub Actions workflow that automatically:

1. Builds and tests the application
2. Creates a Docker image using a secure distroless base
3. Scans the image for vulnerabilities using Trivy
4. Publishes the image to GitHub Container Registry
5. Packages the Helm chart for Kubernetes deployment

## License

MIT License. See [LICENSE](LICENSE) for details.
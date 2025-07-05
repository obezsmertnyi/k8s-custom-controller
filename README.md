# k8s-cli

A Kubernetes custom controller and CLI tool with advanced configuration management, deployment informer, and API server capabilities.

## Features

### Integrated Architecture

The application integrates multiple components into a single binary:

- **Kubernetes Deployment Informer**: Watches for changes in Kubernetes deployments
- **FastHTTP API Server**: Provides HTTP API access to informer data
- **CLI Commands**: For direct interaction with Kubernetes resources

### Configuration Management with Viper

The application uses Viper for flexible configuration management with the following priority order:

1. Command-line flags
2. Environment variables (with `KCUSTOM_` prefix)
3. Configuration file
4. Default values

If a configuration file is explicitly specified with `--config` but not found, the application will exit with an error. If no explicit config file is provided and no default config is found, the application will use default values and environment variables with just a warning message.

#### Configuration File Locations

The application looks for a configuration file named `config.yaml` in these locations:

- Current directory (`./config.yaml`)
- User's home directory (`$HOME/.k8s-custom-controller/config.yaml`)
- System configuration directory (`/etc/k8s-custom-controller/config.yaml`)

You can specify a custom path with the `--config` flag:

```sh
./k8s-cli --config /path/to/my-config.yaml
```

A sample configuration file is provided in the root of the project. See [Configuration Example](#configuration-example) for details.

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

### Integrated FastHTTP API Server

The application includes a high-performance FastHTTP API server with the following features:

- Configurable host and port settings via command-line flags
- Kubernetes informer cache integration for efficient resource queries
- Request logging with unique request IDs and structured metrics
- Multiple resource endpoints (deployments, pods, services, nodes)
- Support for simple and detailed JSON response formats
- Sensible timeout defaults for production use
- 10MB maximum request size limit for security
- Graceful shutdown with signal handling

#### API Endpoints

The server provides the following REST API endpoints:

```
GET /health          - Server health check
GET /deployments     - List Kubernetes deployments (uses informer cache)
GET /pods            - List Kubernetes pods
GET /services        - List Kubernetes services
GET /nodes           - List Kubernetes nodes
```

All resource endpoints support the following query parameters:

- `namespace` - Filter resources by namespace (not applicable to nodes)
- `format=simple` - Return a simple JSON array of resource names instead of detailed information
- Can be disabled via configuration

**Usage:**
```sh
# Start with default API server on port 8080
./k8s-cli

# Start with custom API server port
./k8s-cli --port 8090

# Start with custom host binding
./k8s-cli --host 127.0.0.1

# Disable API server via config
# (In config file, set: kubernetes.disable_api: true)
```

### Deployment Informer

The application includes a Kubernetes deployment informer that:

- Watches for deployment changes (create, update, delete events)
- Caches deployment information for fast API responses
- Configurable resync period and label/field selectors
- Can be disabled via configuration

**Available API Endpoints:**
```
GET /health        # Health check endpoint
GET /deployments   # List all cached deployments
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

## Configuration Example

Below is an example of a complete configuration file (`config.yaml`):

```yaml
kubernetes:
  # Path to kubeconfig file, leave empty for in-cluster config
  kubeconfig: ~/.kube/config
  # Whether to use in-cluster configuration
  in_cluster: false
  # API server queries per second
  qps: 10.0
  # Maximum burst for throttle
  burst: 20
  # Timeout for API server requests
  timeout: 20s
  # Disable informer component if set to true
  disable_informer: false
  # Disable API server component if set to true
  disable_api: false

informer:
  # Namespace to watch, leave empty for all namespaces
  namespace: default
  # Resync period for informer cache
  resync_period: 1m
  # Label selector for filtering deployments
  label_selector: ""
  # Field selector for filtering deployments
  field_selector: ""
  logging:
    # Whether to log informer events
    enable_event_logging: true
    # Log level for informer (trace, debug, info, warn, error)
    log_level: info
  workers:
    # Number of worker goroutines
    count: 2

logging:
  # Global log level (trace, debug, info, warn, error)
  level: info
  # Log format (json, text)
  format: text
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
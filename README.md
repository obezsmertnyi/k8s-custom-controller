# k8s-custom-controller

A Kubernetes custom controller and CLI tool with advanced configuration management.

## Features

### Configuration Management with Viper

The application uses Viper for flexible configuration management with the following priority order:

1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

#### Configuration Options

**Command-line flags:**
```sh
# Set log level
./k8s-cli --log-level debug

# Specify custom config file
./k8s-cli --config /path/to/config.yaml

# Set Kubernetes namespace
./k8s-cli --namespace my-namespace
```

**Environment variables:**
All configuration options can be set using environment variables with the `KCUSTOM_` prefix:
```sh
# Set Kubernetes namespace
export KCUSTOM_KUBERNETES_NAMESPACE=test-namespace

# Set log level
export KCUSTOM_LOGGING_LEVEL=debug

# Set log format
export KCUSTOM_LOGGING_FORMAT=json

# Set Kubernetes QPS and burst
export KCUSTOM_KUBERNETES_QPS=100
export KCUSTOM_KUBERNETES_BURST=200

# Run the application
./k8s-cli
```

**Configuration file:**
The application looks for a `config.yaml` file in the following locations:
- Current directory
- `$HOME/.k8s-custom-controller/`
- `/etc/k8s-custom-controller/`

Example configuration file:
```yaml
kubernetes:
  kubeconfig: ~/.kube/config
  namespace: default
  in_cluster: false
  timeout: 30s
  qps: 50
  burst: 100
logging:
  level: info
  format: text
```

**View current configuration:**
```sh
./k8s-cli config view
```

### Log Level Support

The application supports different log levels using `zerolog`:

```sh
# Available log levels
./k8s-cli --log-level trace  # Most verbose, includes all logs
./k8s-cli --log-level debug  # Detailed debugging information
./k8s-cli --log-level info   # Default level, general operational information
./k8s-cli --log-level warn   # Warning conditions
./k8s-cli --log-level error  # Error conditions
```

Log level can also be set via configuration file or environment variable:
```sh
export KCUSTOM_LOGGING_LEVEL=debug
```

### Log Format

Two log formats are supported:
- `text` (default): Human-readable format with colors and formatting
- `json`: Structured JSON format for machine processing

Set the format in the configuration file:
```yaml
logging:
  format: json
```

Or via environment variable:
```sh
export KCUSTOM_LOGGING_FORMAT=json
```

## Testing

### Configuration Tests

The project includes tests for configuration management:

```sh
# Run configuration tests
go test ./tests
```

These tests verify:
- Default configuration values
- Environment variable overrides
- Configuration file loading

### Manual Configuration Testing

A test script is provided to manually test configuration loading from different sources:

```sh
# Run the configuration test script
./scripts/test-config.sh
```

This script tests:
1. Default configuration
2. Configuration from environment variables
3. Configuration from config file
4. Configuration from command line flags

## Project Structure

- `cmd/` — Contains your CLI commands and configuration management
  - `root.go` — Root command and logger configuration
  - `config.go` — Configuration management with Viper
- `main.go` — Entry point for your application

## License

MIT License. See [LICENSE](LICENSE) for details. 
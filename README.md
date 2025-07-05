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

### FastHTTP Server Command

The application includes a high-performance FastHTTP server with the following features:

- Configurable host and port settings
- Request logging with detailed metrics
- Sensible timeout defaults for production use
- 10MB maximum request size limit for security

**Usage:**
```sh
# Start server on localhost:8080 (default)
./k8s-cli server

# Start server on all interfaces with custom port
./k8s-cli server --host 0.0.0.0 --port 8090

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
- `main.go` — Entry point for the application
- `scripts/` — Test and utility scripts
- `tests/` — Test files

## Testing

```sh
# Run all tests
go test ./...

# Run specific test files
go test ./tests/server_test.go

# Manual configuration testing
./scripts/test-config.sh
```

## License

MIT License. See [LICENSE](LICENSE) for details.

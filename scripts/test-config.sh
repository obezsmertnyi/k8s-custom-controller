#!/bin/bash

# Build the application
echo "Building application..."
go build -o k8s-cli

# Test default configuration
echo "\n=== Testing default configuration ==="
./k8s-cli config view

# Test environment variables
echo "\n=== Testing environment variables ==="
export KCUSTOM_KUBERNETES_NAMESPACE="env-namespace"
export KCUSTOM_LOGGING_LEVEL="debug"
export KCUSTOM_LOGGING_FORMAT="json"
./k8s-cli config view

# Create a test config file
echo "\n=== Creating test config file ==="
cat > config.yaml << EOF
kubernetes:
  namespace: file-namespace
  qps: 100
  burst: 200
logging:
  level: trace
  format: json
EOF

# Test config file
echo "\n=== Testing config file ==="
./k8s-cli config view

# Test command line flags
echo "\n=== Testing command line flags ==="
./k8s-cli config view --namespace="flag-namespace" --log-level="warn"

# Clean up
rm config.yaml
unset KCUSTOM_KUBERNETES_NAMESPACE
unset KCUSTOM_LOGGING_LEVEL
unset KCUSTOM_LOGGING_FORMAT

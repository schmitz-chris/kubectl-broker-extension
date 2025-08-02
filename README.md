# kubectl-broker

A kubectl plugin for streamlined health diagnostics of HiveMQ broker clusters running on Kubernetes.

## Features

- 🚀 **Single Pod Health Checks**: Check individual HiveMQ broker pods
- 🔄 **Parallel Cluster Health Checks**: Concurrent health checks across entire StatefulSets
- 🔍 **Automatic Discovery**: Find HiveMQ brokers across all accessible namespaces
- 🎯 **Intelligent Defaults**: Automatically uses StatefulSet "broker" and current kubectl context namespace
- 📊 **Professional Output**: Clean tabular results with response times and status details
- 🛡️ **Robust Error Handling**: Comprehensive error messages with actionable guidance
- 🔧 **Port Discovery**: Automatic health port detection with manual override support
- 🌐 **kubie Integration**: Full compatibility with kubie context manager

## Installation

### Automatic Installation (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/your-repo/kubectl-broker/main/install.sh | bash
```

### Manual Installation

1. **Download or Build**:
   ```bash
   # Option 1: Download from releases
   wget https://github.com/your-repo/kubectl-broker/releases/latest/download/kubectl-broker
   
   # Option 2: Build from source
   git clone https://github.com/your-repo/kubectl-broker.git
   cd kubectl-broker
   go build -o kubectl-broker ./cmd/kubectl-broker
   ```

2. **Install as kubectl plugin**:
   ```bash
   # Create installation directory
   mkdir -p ~/.kubectl-broker
   
   # Copy binary
   cp kubectl-broker ~/.kubectl-broker/
   chmod +x ~/.kubectl-broker/kubectl-broker
   
   # Add to PATH (choose your shell)
   echo 'export PATH="$HOME/.kubectl-broker:$PATH"' >> ~/.bashrc   # For bash
   echo 'export PATH="$HOME/.kubectl-broker:$PATH"' >> ~/.zshrc    # For zsh
   
   # Reload shell
   source ~/.bashrc  # or ~/.zshrc
   ```

3. **Verify Installation**:
   ```bash
   kubectl plugin list | grep broker
   kubectl broker --help
   ```

## Usage

### As kubectl Plugin

Once installed, use `kubectl broker` instead of `kubectl-broker`:

```bash
# Discovery mode - find all HiveMQ brokers
kubectl broker --discover

# Quick cluster health check with intelligent defaults
kubectl broker

# Single pod health check
kubectl broker --pod broker-0 --namespace my-hivemq-namespace

# Full cluster health check (explicit)
kubectl broker --statefulset broker --namespace my-hivemq-namespace

# With custom port
kubectl broker --statefulset broker --namespace my-hivemq-namespace --port 9090
```

### Intelligent Defaults

kubectl-broker includes smart defaults for common usage patterns:

```bash
# Automatically uses StatefulSet "broker" and current kubectl context namespace
kubectl broker

# Equivalent to:
kubectl broker --statefulset broker --namespace $(kubectl config view --minify -o jsonpath='{..namespace}')

# Visual feedback shows which defaults were applied
# 🎯 Using default StatefulSet: broker
# 🎯 Using namespace from context: my-namespace
```

### Direct Binary Usage

You can also run the binary directly:

```bash
./kubectl-broker --discover
./kubectl-broker  # Uses intelligent defaults
./kubectl-broker --pod broker-0 --namespace my-hivemq-namespace
./kubectl-broker --statefulset broker --namespace my-hivemq-namespace
```

## Command Examples

### Quick Health Check with Defaults
```bash
kubectl broker
```
Output:
```
🎯 Using default StatefulSet: broker
🎯 Using namespace from context: production-hivemq
Checking health of StatefulSet broker in namespace production-hivemq
Found 3 pods in StatefulSet

[Health check results...]
```

### Discovery Mode
```bash
kubectl broker --discover
```
Output:
```
Namespace: production-hivemq
  - broker-0
  - broker-1
  Single pod: kubectl broker --pod broker-0 --namespace production-hivemq
  All pods:   kubectl broker --statefulset broker --namespace production-hivemq
```

### Cluster Health Check (Explicit)
```bash
kubectl broker --statefulset broker --namespace production-hivemq
```
Output:
```
Starting concurrent health checks for 3 pods...

POD NAME  STATUS   HEALTH PORT  LOCAL PORT  RESPONSE TIME  DETAILS
--------  ------   -----------  ----------  -------------  -------
broker-0  HEALTHY  9090         51411       145ms          Health check successful
broker-1  HEALTHY  9090         51412       150ms          Health check successful
broker-2  HEALTHY  9090         51410       127ms          Health check successful

Summary: 3/3 pods healthy
✅ All pods are healthy!
```

## Command Line Flags

| Flag | Description | Required | Example |
|------|-------------|----------|---------|
| `--discover` | Discover available broker pods and namespaces | No | `kubectl broker --discover` |
| `--pod` | Name of specific pod to check (single pod mode) | Optional* | `--pod broker-0` |
| `--statefulset` | Name of StatefulSet to check (cluster mode) | Optional* | `--statefulset broker` |
| `--namespace, -n` | Kubernetes namespace | Optional** | `--namespace production` |
| `--port, -p` | Manual port override for health checks | No | `--port 9090` |

*If neither `--pod` nor `--statefulset` is specified, defaults to `--statefulset broker`  
**Defaults to current kubectl context namespace. Not required when using `--discover`

## Architecture

kubectl-broker is designed specifically for HiveMQ broker clusters where:
- Brokers run as StatefulSets in Kubernetes
- Each pod exposes a health API endpoint (typically on port 9090 named "health")
- Health checks return JSON status information about cluster state, extensions, and MQTT listeners
- Multiple broker instances need to be checked individually for complete cluster health status

The tool uses:
- **Native Kubernetes Integration**: `k8s.io/client-go` library for robust API access
- **Concurrent Processing**: Goroutines with dynamic port allocation for parallel health checks
- **Port Forwarding**: Automated port-forwarding to bypass network policies
- **Smart Discovery**: Label selectors to find pods belonging to StatefulSets

## Troubleshooting

### Plugin Not Found
```bash
# Check if binary is in PATH
which kubectl-broker

# List all kubectl plugins
kubectl plugin list

# Manually add to PATH
export PATH="$HOME/.kubectl-broker:$PATH"
```

### Permission Errors
```bash
# Check kubeconfig permissions
kubectl auth can-i list pods --namespace your-namespace
kubectl auth can-i create pods/portforward --namespace your-namespace
```

### Port Discovery Issues
If automatic port discovery fails, use manual override:
```bash
kubectl broker --statefulset broker --namespace your-namespace --port 9090
```

### Connection Issues
1. Verify pods are running: `kubectl get pods -n your-namespace`
2. Check pod readiness: `kubectl describe pod broker-0 -n your-namespace`
3. Test port-forward manually: `kubectl port-forward broker-0 9090:9090 -n your-namespace`

## Development

### Building from Source
```bash
git clone https://github.com/your-repo/kubectl-broker.git
cd kubectl-broker
go mod download
go build -o kubectl-broker ./cmd/kubectl-broker
```

### Running Tests
```bash
go test ./...
```

### Project Structure
```
kubectl-broker/
├── cmd/kubectl-broker/    # Main application
├── pkg/                   # Core packages
│   ├── concurrent.go      # Parallel health checking
│   ├── discovery.go       # Pod/StatefulSet discovery
│   ├── errors.go          # Error handling
│   ├── k8s.go            # Kubernetes client
│   └── portforward.go     # Port forwarding logic
├── install.sh            # Installation script
└── README.md
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and add tests
4. Commit your changes: `git commit -am 'Add feature'`
5. Push to the branch: `git push origin feature-name`
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📝 [Report Issues](https://github.com/your-repo/kubectl-broker/issues)
- 💬 [Discussions](https://github.com/your-repo/kubectl-broker/discussions)
- 📖 [Documentation](https://github.com/your-repo/kubectl-broker/wiki)
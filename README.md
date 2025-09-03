# kubectl-broker

A production-ready kubectl plugin for comprehensive HiveMQ cluster management on Kubernetes, providing health diagnostics, backup operations, and intelligent cluster monitoring.

## Features

### Health Diagnostics
- **Single Pod Health Checks**: Check individual HiveMQ broker pods with detailed component analysis
- **Parallel Cluster Health Checks**: Concurrent health checks across entire StatefulSets
- **Enhanced Health API Analysis**: Comprehensive JSON parsing with component-level status (cluster, extensions, MQTT listeners)
- **Individual Extension Details**: Detailed information for each HiveMQ extension including version and license status
- **Color-Coded Status Display**: Visual health indicators (UP/DOWN/DEGRADED) for improved monitoring
- **Multiple Output Formats**: Tabular, JSON, raw, and detailed component breakdown
- **Automatic Discovery**: Find HiveMQ brokers across all accessible namespaces
- **Intelligent Defaults**: Automatically uses StatefulSet "broker" and current kubectl context namespace

### Backup Management
- **Backup Operations**: Create, list, download, and monitor backup status
- **Backup Directory Management**: Automatic backup directory moving within pod filesystem
- **Progress Monitoring**: Real-time status polling with progress indicators
- **Authentication Support**: Username/password authentication for secured HiveMQ instances
- **File Download**: Automatic backup download with progress bars

## Installation

### Automatic Installation (Recommended)

```bash
# Build and install from source with automated setup
make install-auto
```

### Manual Installation

1. **Build from source**:
   ```bash
   git clone <repository-url>
   cd kubectl-broker-extension
   make build-small  # Optimized 35MB build
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

### Command Structure

kubectl-broker uses a subcommand architecture for different operations:

```bash
# Show available commands
kubectl broker

# Health diagnostics
kubectl broker status [options]

# Backup management  
kubectl broker backup [subcommand] [options]
```

### Health Diagnostics (`status` subcommand)

```bash
# Simple usage with intelligent defaults
kubectl broker status

# Discovery mode - find all HiveMQ brokers
kubectl broker status --discover

# Single pod health check
kubectl broker status --pod broker-0 --namespace my-hivemq-namespace

# Full cluster health check (explicit)
kubectl broker status --statefulset broker --namespace my-hivemq-namespace

# Enhanced output formats
kubectl broker status --json                    # Raw JSON for external tools
kubectl broker status --detailed                # Component breakdown + debug info
kubectl broker status --endpoint liveness       # Specific health endpoint
kubectl broker status --raw                     # Unprocessed response

# With custom port
kubectl broker status --statefulset broker --namespace my-hivemq-namespace --port 9090
```

### Backup Management (`backup` subcommand)

```bash
# Create new backup
kubectl broker backup create --statefulset broker --namespace production

# List all backups
kubectl broker backup list --statefulset broker --namespace production

# Download specific backup
kubectl broker backup download --id abc123 --output-dir ./backups

# Download latest backup
kubectl broker backup download --latest --output-dir ./backups

# Check backup status
kubectl broker backup status --id abc123
kubectl broker backup status --latest

# With authentication (for secured HiveMQ instances)
kubectl broker backup create --username admin --password secret

# Move backup to different directory within pod
kubectl broker backup create --destination /opt/hivemq/data/backup
```

### Intelligent Defaults

kubectl-broker includes smart defaults for common usage patterns:

```bash
# Automatically uses StatefulSet "broker" and current kubectl context namespace
kubectl broker status

# Equivalent to:
kubectl broker status --statefulset broker --namespace $(kubectl config view --minify -o jsonpath='{..namespace}')

# Visual feedback shows which defaults were applied
# Using default StatefulSet: broker
# Using namespace from context: my-namespace
```

### Direct Binary Usage

You can also run the binary directly:

```bash
./kubectl-broker status --discover
./kubectl-broker status  # Uses intelligent defaults
./kubectl-broker status --pod broker-0 --namespace my-hivemq-namespace
./kubectl-broker backup create --statefulset broker --namespace my-hivemq-namespace
```

## Command Examples

### Health Diagnostics Examples

#### Quick Health Check with Defaults
```bash
kubectl broker status
```
Output:
```
Using default StatefulSet: broker
Using namespace from context: production-hivemq
Checking health of StatefulSet broker in namespace production-hivemq
Found 3 pods in StatefulSet

POD NAME  STATUS   DETAILS
--------  ------   -------
broker-0  HEALTHY  Overall: [UP], Components: 8 total, 8 healthy
broker-1  HEALTHY  Overall: [UP], Components: 8 total, 8 healthy  
broker-2  HEALTHY  Overall: [UP], Components: 8 total, 8 healthy

Summary: 3/3 pods healthy
```

#### Discovery Mode
```bash
kubectl broker status --discover
```
Output:
```
Namespace: production-hivemq
  - broker-0
  - broker-1
  Single pod: kubectl broker status --pod broker-0 --namespace production-hivemq
  All pods:   kubectl broker status --statefulset broker --namespace production-hivemq
```

#### Detailed Health Check Output
```bash
kubectl broker status --statefulset broker --namespace production-hivemq --detailed
```
Output:
```
Starting concurrent health checks for 3 pods...

POD NAME  STATUS   HEALTH PORT  LOCAL PORT  RESPONSE TIME  DETAILS
--------  ------   -----------  ----------  -------------  -------
broker-0  HEALTHY  9090         51411       145ms          Overall: [UP], Components: 8 total, 8 healthy
broker-1  HEALTHY  9090         51412       150ms          Overall: [UP], Components: 8 total, 8 healthy
broker-2  HEALTHY  9090         51410       127ms          Overall: [UP], Components: 8 total, 8 healthy

Summary: 3/3 pods healthy

Pod: broker-0
Overall Health: [UP]
Components:
  - cluster: [UP] (cluster-id: 2FVes, cluster-nodes: [dZIGZ fuD1n])
  - extensions: [UP]
  - mqtt-listeners: [UP]
  - control-center: [UP]
  - rest-api: [UP]
```

### Backup Management Examples

#### Create Backup
```bash
kubectl broker backup create --statefulset broker --namespace production
```
Output:
```
Creating backup for StatefulSet broker in namespace production
Backup created: 20250819-143025
Waiting for completion.... done

Backup ID: 20250819-143025
Status: COMPLETED
Size: 1.2 MB | Created: 2025-08-19T14:30:25Z
```

#### List Backups
```bash
kubectl broker backup list --statefulset broker --namespace production
```
Output:
```
BACKUP ID        STATUS      SIZE     CREATED
---------        ------      ----     -------
20250819-143025  COMPLETED   1.2 MB   2025-08-19T14:30:25Z
20250819-120030  COMPLETED   1.1 MB   2025-08-19T12:00:30Z
20250818-180015  COMPLETED   1.0 MB   2025-08-18T18:00:15Z

Total: 3 backups
```

#### Download Backup
```bash
kubectl broker backup download --latest --output-dir ./backups --statefulset broker --namespace production
```
Output:
```
Downloading latest backup...
Progress: [████████████████████████████████] 100% (1.2 MB / 1.2 MB)

Download completed!
File: ./backups/hivemq-backup-20250819-143025.zip
Size: 1.2 MB
```

## Command Line Reference

### Global Flags

| Flag              | Description                     | Example                     |
|-------------------|---------------------------------|-----------------------------|
| `--help, -h`      | Show help information           | `kubectl broker --help`     |

### Status Subcommand Flags

| Flag              | Description                                     | Required   | Example                               |
|-------------------|-------------------------------------------------|------------|---------------------------------------|
| `--discover`      | Discover available broker pods and namespaces   | No         | `kubectl broker status --discover`    |
| `--pod`           | Name of specific pod to check (single pod mode) | Optional*  | `--pod broker-0`                      |
| `--statefulset`   | Name of StatefulSet to check (cluster mode)     | Optional*  | `--statefulset broker`                |
| `--namespace, -n` | Kubernetes namespace                            | Optional** | `--namespace production`              |
| `--port, -p`      | Manual port override for health checks          | No         | `--port 9090`                         |
| `--json`          | Output raw JSON response for external tools      | No         | `kubectl broker status --json`        |
| `--detailed`      | Show detailed component breakdown + debug info   | No         | `kubectl broker status --detailed`    |
| `--raw`           | Show unprocessed response                       | No         | `kubectl broker status --raw`         |
| `--endpoint`      | Specific health endpoint (health/liveness/readiness) | No    | `--endpoint liveness`                 |

### Backup Subcommand Flags

#### Create Backup
| Flag              | Description                                     | Required   | Example                               |
|-------------------|-------------------------------------------------|------------|---------------------------------------|
| `--statefulset`   | Name of StatefulSet containing broker           | Optional*  | `--statefulset broker`                |
| `--namespace, -n` | Kubernetes namespace                            | Optional** | `--namespace production`              |
| `--username`      | Username for HiveMQ authentication              | No         | `--username admin`                    |
| `--password`      | Password for HiveMQ authentication              | No         | `--password secret`                   |
| `--destination`   | Move backup to specific directory within pod    | No         | `--destination /opt/hivemq/data/backup` |

#### List Backups
| Flag              | Description                                     | Required   | Example                               |
|-------------------|-------------------------------------------------|------------|---------------------------------------|
| `--statefulset`   | Name of StatefulSet containing broker           | Optional*  | `--statefulset broker`                |
| `--namespace, -n` | Kubernetes namespace                            | Optional** | `--namespace production`              |
| `--username`      | Username for HiveMQ authentication              | No         | `--username admin`                    |
| `--password`      | Password for HiveMQ authentication              | No         | `--password secret`                   |

#### Download Backup
| Flag              | Description                                     | Required   | Example                               |
|-------------------|-------------------------------------------------|------------|---------------------------------------|
| `--id`            | Specific backup ID to download                  | Optional*** | `--id 20250819-143025`               |
| `--latest`        | Download latest backup                          | Optional*** | `--latest`                           |
| `--output-dir`    | Local directory to save backup file            | Yes        | `--output-dir ./backups`              |
| `--statefulset`   | Name of StatefulSet containing broker           | Optional*  | `--statefulset broker`                |
| `--namespace, -n` | Kubernetes namespace                            | Optional** | `--namespace production`              |
| `--username`      | Username for HiveMQ authentication              | No         | `--username admin`                    |
| `--password`      | Password for HiveMQ authentication              | No         | `--password secret`                   |

#### Check Backup Status
| Flag              | Description                                     | Required   | Example                               |
|-------------------|-------------------------------------------------|------------|---------------------------------------|
| `--id`            | Specific backup ID to check                     | Optional*** | `--id 20250819-143025`               |
| `--latest`        | Check status of latest backup                   | Optional*** | `--latest`                           |
| `--statefulset`   | Name of StatefulSet containing broker           | Optional*  | `--statefulset broker`                |
| `--namespace, -n` | Kubernetes namespace                            | Optional** | `--namespace production`              |
| `--username`      | Username for HiveMQ authentication              | No         | `--username admin`                    |
| `--password`      | Password for HiveMQ authentication              | No         | `--password secret`                   |

### Notes
*If not specified, defaults to `broker`  
**Defaults to current kubectl context namespace  
***Either `--id` or `--latest` must be specified

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
kubectl broker status --statefulset broker --namespace your-namespace --port 9090
```

### Connection Issues
1. Verify pods are running: `kubectl get pods -n your-namespace`
2. Check pod readiness: `kubectl describe pod broker-0 -n your-namespace`
3. Test port-forward manually: `kubectl port-forward broker-0 9090:9090 -n your-namespace`

## Development

### Building from Source

This project uses a professional build system with Make:

```bash
git clone <repository-url>
cd kubectl-broker-extension
go mod download

# Build commands (choose appropriate for your needs)
make build          # Standard build
make build-small    # Optimized build (35MB vs 53MB)
make release        # Release build with optimizations

# Installation  
make install        # Install as kubectl plugin (standard build)
make install-small  # Install as kubectl plugin (optimized build)
make install-auto   # Install with automatic PATH setup

# Development
make dev           # Build with race detector
make test          # Test basic functionality
make check         # Run all code quality checks (fmt, vet, test-go)

# Direct Go commands (also supported)
go run cmd/kubectl-broker/main.go    # Run directly during development
go fmt ./...                         # Format code
go vet ./...                         # Static analysis

# Maintenance
make clean         # Remove build artifacts
make uninstall     # Remove installed plugin
```

### Running Tests
```bash
make test
# or
go test ./...
```

### Project Structure
```
kubectl-broker/
├── cmd/kubectl-broker/       # Main application with subcommand architecture
│   ├── main.go              # Root command and CLI entry point
│   ├── status.go            # Health diagnostics subcommand
│   └── backup.go            # Backup management subcommand
├── pkg/                     # Core functionality packages
│   ├── k8s.go              # Kubernetes client (optimized with typed clients)
│   ├── health/             # HiveMQ Health API parsing and analysis
│   └── backup/             # HiveMQ backup operations and REST API client
│       ├── client.go       # REST API client for backup operations
│       ├── operations.go   # Backup CRUD operations
│       └── types.go        # Data structures and response types
├── Makefile                # Professional build system with optimization
├── install.sh              # Automated kubectl plugin installation script
├── CLAUDE.md               # Development guidance for Claude Code
└── README.md               # User documentation
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and add tests
4. Commit your changes: `git commit -am 'Add feature'`
5. Push to the branch: `git push origin feature-name`
6. Submit a pull request

## Support

For support and feedback, please refer to the project documentation or contact the development team.

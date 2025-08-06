# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a production-ready Go project for `kubectl-broker`, a kubectl plugin CLI tool that provides comprehensive HiveMQ cluster management for Kubernetes. The project has completed all major phases (1-7) and features intelligent defaults, concurrent health checks, optimized binary size, enhanced HiveMQ Health API analysis, extensible subcommand architecture, and complete backup management functionality.

## Project Structure

- `cmd/kubectl-broker/main.go` - Root command with subcommand architecture (Phase 6)
- `cmd/kubectl-broker/status.go` - Health diagnostics subcommand with intelligent defaults
- `cmd/kubectl-broker/backup.go` - Complete backup operations with subcommands (Phase 7)
- `pkg/` - Core functionality packages (k8s client, port-forwarding, concurrent health checks)
- `pkg/health/` - HiveMQ Health API parsing and analysis (Phase 5)
- `pkg/backup/` - HiveMQ backup operations and REST API client (Phase 7)
- `PLAN.md` - Implementation roadmap (all phases completed including Phase 7)
- `OBJECTS.md` - Example Kubernetes objects and HiveMQ health API responses
- `Makefile` - Professional build system with size optimization targets
- `install.sh` - Automated kubectl plugin installation script
- `README.md` - Complete user documentation
- `.claude/settings.local.json` - Local permissions configuration for Claude Code

## Development Commands

This is a Go project with a professional build system using Make:

```bash
# Build commands (choose appropriate for your needs)
make build          # Standard build
make build-small    # Optimized build (35MB vs 53MB)
make build-upx      # UPX compressed (Linux only)
make release        # Release build with optimizations

# Installation
make install        # Install as kubectl plugin
make install-auto   # Install with automatic PATH setup

# Development 
make dev           # Build with race detector
make test          # Test basic functionality
make check         # Run all code quality checks (fmt, vet, test)

# Direct Go commands (also supported)
go run cmd/kubectl-broker/main.go    # Run directly during development
go fmt ./...                         # Format code
go vet ./...                         # Static analysis
gofmt -s -w .                       # Standard formatting

# Maintenance
make clean         # Remove build artifacts  
make uninstall     # Remove installed plugin
```

The tool is used as a kubectl plugin with subcommand architecture:
```bash
# Show available commands
kubectl broker

# Health diagnostics (Phase 6 subcommand structure)
kubectl broker status                                    # Simple usage with intelligent defaults
kubectl broker status --discover                         # Discovery mode
kubectl broker status --statefulset broker --namespace production
kubectl broker status --pod broker-0 --namespace production

# Enhanced health API analysis (Phase 5) with color-coded status indicators
kubectl broker status --json                             # Raw JSON output for external tools (colors disabled)
kubectl broker status --detailed                         # Detailed component breakdown + debug info (colors enabled)
kubectl broker status --endpoint liveness                # Specific health endpoint with colored status
kubectl broker status --statefulset broker --raw         # Unprocessed response (colors disabled)
kubectl broker status --pod broker-0 --endpoint readiness # Readiness check with colored indicators

# Backup management (Phase 7 complete functionality)
kubectl broker backup create --statefulset broker --namespace production  # Create new backup
kubectl broker backup list --statefulset broker --namespace production     # List all backups
kubectl broker backup download --id abc123 --output-dir ./backups          # Download specific backup
kubectl broker backup download --latest --output-dir ./backups             # Download latest backup
kubectl broker backup status --id abc123                                   # Check backup status
kubectl broker backup status --latest                                      # Check latest backup status

# With authentication (optional)
kubectl broker backup create --username admin --password secret
```

## Architecture

The tool has completed all planned development phases:

### ✅ Phase 1: Single Pod Connection (Completed)
- Native `k8s.io/client-go` library with optimized typed clients
- Enhanced port discovery with fallback strategy and `--port/-p` flag override  
- Programmatic port-forwarding to query health endpoints directly
- Single pod targeting via `--pod` and `--namespace` flags
- Comprehensive error handling for RBAC issues, pod status, and connectivity problems

### ✅ Phase 2: Parallel Cluster Health Checks (Completed)
- StatefulSet pod discovery using label selectors
- Dynamic port allocation using `net.Listen(":0")` for concurrent connections
- Concurrent health checks across multiple broker instances with goroutines
- Enhanced error handling for StatefulSet discovery and individual pod failures
- Professional tabular output with response times and status details

### ✅ Phase 3: Production Polish (Completed)
- Intelligent defaults (namespace from kubectl context, default StatefulSet name "broker")
- Visual feedback showing which defaults were applied (🎯 indicators)
- Comprehensive error handling with actionable guidance for users
- Context-aware error messages that suggest solutions

### ✅ Phase 4: kubectl Plugin Integration (Completed)
- Professional installation as kubectl plugin with `~/.kubectl-broker/` directory
- Automated installation script with shell detection and PATH management
- Cross-platform build system with size optimization

### ✅ Phase 5: Enhanced HiveMQ Health API Analysis (Completed)
- Advanced JSON parsing of HiveMQ health responses with Go structs
- Multiple output formats: tabular, JSON, raw, detailed component breakdown
- Support for different health endpoints: health, liveness, readiness
- Rich diagnostic information showing component-level health status
- External tool integration with JSON output for monitoring pipelines
- Clean minimal output by default, verbose debug info only with `--detailed` flag
- **Color-coded health status indicators** for improved visual recognition of broker states

### ✅ Phase 6: Subcommand Architecture (Completed)
- Extensible CLI structure with parent command and subcommands
- `status` subcommand containing all health checking functionality
- `backup` subcommand framework for future backup operations
- Professional command structure following kubectl plugin patterns
- Maintains backward compatibility through clear command separation
- Foundation for additional HiveMQ cluster management features

### ✅ Phase 7: HiveMQ Backup Management (Completed)
- Complete backup management system using HiveMQ REST API
- Four backup subcommands: create, list, download, status
- Intelligent defaults and consistent UX with existing health monitoring
- Progress indicators and status polling for long-running operations
- File download with progress bars and automatic filename handling
- Color-coded status display matching health command patterns
- Comprehensive error handling with actionable guidance
- Authentication support for secured HiveMQ instances

### 🚀 Binary Size Optimization (Completed)
- Optimized from 53MB to 35MB (-34% reduction) using selective Kubernetes client imports
- Replaced full `kubernetes.Clientset` with specific typed clients (`CoreV1Client`, `AppsV1Client`)
- Advanced build optimization with `-ldflags="-w -s"`, `-trimpath`, `CGO_ENABLED=0`
- UPX compression support for Linux systems

## Key Technical Decisions

- **Optimized Kubernetes Integration**: Uses specific typed clients (`CoreV1Client`, `AppsV1Client`) instead of full `kubernetes.Clientset` for minimal binary size
- **Programmatic Port-Forwarding**: Self-contained operation using custom REST client for SPDY connections
- **Concurrent Architecture**: Goroutines with dynamic port allocation using centralized `GetRandomPort()` utility
- **Intelligent Defaults**: Context-aware namespace detection and StatefulSet name defaulting
- **User-Centric Installation**: `~/.kubectl-broker/` directory avoiding system-wide installation
- **HiveMQ-Specific**: Targets broker StatefulSets with health endpoints on port named "health"
- **Local Operator Focus**: Designed for diagnostic execution from operator machines
- **Enhanced Health Analysis**: Comprehensive JSON parsing with multiple output formats for both human operators and external tool integration
- **Color-Coded Status Display**: Visual health status indicators using `github.com/fatih/color` for improved user experience
- **Production Code Quality**: Clean, linter-compliant Go code following best practices with proper error handling and deprecation management

## HiveMQ Integration

The tool is specifically designed for HiveMQ broker clusters where:
- Brokers run as StatefulSets in Kubernetes
- Each pod exposes a health API endpoint (typically on port 9090)
- Health checks return JSON status information about cluster state, extensions, and MQTT listeners
- Multiple broker instances need to be checked individually to get complete cluster health status

### Enhanced Health API Support (Phase 5)

The tool now provides comprehensive analysis of HiveMQ's health API responses:

- **Component Analysis**: Parses and displays status of individual components (cluster, MQTT, extensions, control-center, rest-api, etc.)
- **Multiple Endpoints**: Supports `/api/v1/health/` (full), `/api/v1/health/liveness` (basic), `/api/v1/health/readiness` (ready to serve)
- **Status Interpretation**: Understands HiveMQ status values (UP, DOWN, DEGRADED, UNKNOWN, OUT_OF_SERVICE)
- **Output Formats**: 
  - **Tabular**: Enhanced with component counts (e.g., "Overall: [UP], Components: 8 total, 8 healthy")
  - **JSON**: Raw responses for external tool integration (jq, monitoring systems)
  - **Detailed**: Component-by-component breakdown with details
  - **Raw**: Unprocessed responses for debugging

Example clean output (normal usage with color-coded status):
```
POD NAME  STATUS   DETAILS
--------  ------   -------
broker-0  HEALTHY  Overall: [UP], Components: 8 total, 8 healthy    # [UP] shown in green
broker-1  HEALTHY  Overall: [UP], Components: 8 total, 8 healthy    # [UP] shown in green

Summary: 2/2 pods healthy
```

**Color Scheme:**
- `[UP]`: Green + Bold (healthy status)
- `[DOWN]`: Red + Bold (critical/failed status)  
- `[DEGRADED]`: Yellow + Bold (warning/degraded status)
- `[UNKNOWN]`: White (unclear status)
- `[OUT_OF_SERVICE]`: Magenta (intentionally offline)

Example detailed output (with `--detailed` flag):
```
Using default kubeconfig: /Users/chris/.kube/config
Using cluster: arn:aws:eks:eu-central-1:...
[... all debug information ...]

POD NAME  STATUS   HEALTH PORT  LOCAL PORT  RESPONSE TIME  DETAILS
--------  ------   -----------  ----------  -------------  -------
broker-0  HEALTHY  9090         54740       130ms          Overall: [UP], Components: 8 total, 8 healthy
broker-1  HEALTHY  9090         54741       113ms          Overall: [UP], Components: 8 total, 8 healthy

Summary: 2/2 pods healthy

Pod: broker-0
Overall Health: [UP]
Components:
  - cluster: [UP] (cluster-id: 2FVes, cluster-nodes: [dZIGZ fuD1n], ...)
  - extensions: [UP]
  [... detailed component breakdown ...]
```
```

## CLI Coding Guidelines

- **Emojis**: Do not use emojis for CLI output in this application

## Recent Updates (2025-01-06)

### Professional CLI Output
- **No Emojis**: All CLI output uses clean, professional text without decorative emojis
- **Streamlined Backup Creation**: Reduced verbose output from 13 lines to 6 lines, eliminated duplicate messages
- **Better Error Messages**: Improved download error handling with clear explanations when functionality is not supported

### Backup Functionality Fixes
- **JSON Parsing**: Fixed backup response parsing to handle HiveMQ's `{"backup": {...}}` wrapper format
- **API Endpoints**: Corrected backup status endpoint from `/backups/{id}/status` to `/backups/{id}`
- **Field Mapping**: Updated JSON field names to match HiveMQ API (`state` not `status`, `bytes` not `size`, `items` not `backups`)
- **Download Handling**: Added intelligent download endpoint detection and informative error messages for unsupported download functionality

### Output Examples

#### Backup Creation (Before/After)
**Before (verbose, 13 lines):**
```
Warning: Backup operations may consume significant disk space...
Creating backup using service hivemq-broker-api...
Connected to API on port 51635, testing management API...
Management API available, initiating backup...
Backup created with ID: 20250806-081834
Waiting for backup to complete...
Status: COMPLETED
Backup completed successfully! Size: 1.0 KB    [duplicate]
Backup completed successfully!                  [duplicate]
...
```

**After (clean, 6 lines):**
```
Creating backup for StatefulSet broker in namespace xyz
Backup created: 20250806-083755
Waiting for completion.... done

Backup ID: 20250806-083755
Status: COMPLETED
Size: 1.0 KB | Created: 2025-08-06T08:37:55Z
```

#### Error Messages (Improved)
**Before:** `HTTP 404: {"errors":[{"title":"Resource not found"}]}`

**After:** `backup download not supported: all download endpoints returned 404. This HiveMQ instance (version 4.x) may not have backup download functionality enabled or available. You can create and list backups, but downloading them may not be supported in this configuration.`
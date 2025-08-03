# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a production-ready Go project for `kubectl-broker`, a kubectl plugin CLI tool that streamlines health diagnostics for HiveMQ clusters running on Kubernetes. The project has completed all major phases (1-5) and is fully functional with intelligent defaults, concurrent health checks, optimized binary size, and enhanced HiveMQ Health API analysis.

## Project Structure

- `cmd/kubectl-broker/main.go` - Main CLI application with intelligent defaults and enhanced health options
- `pkg/` - Core functionality packages (k8s client, port-forwarding, concurrent health checks)
- `pkg/health/` - HiveMQ Health API parsing and analysis (Phase 5)
- `PLAN.md` - Implementation roadmap (all phases completed including Phase 5)
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

# Maintenance
make clean         # Remove build artifacts  
make uninstall     # Remove installed plugin
```

The tool is used as a kubectl plugin:
```bash
# Simple usage with intelligent defaults
kubectl broker

# Discovery mode
kubectl broker --discover

# Explicit usage
kubectl broker --statefulset broker --namespace production
kubectl broker --pod broker-0 --namespace production

# Enhanced health API analysis (Phase 5) with color-coded status indicators
kubectl broker --json                              # Raw JSON output for external tools (colors disabled)
kubectl broker --detailed                          # Detailed component breakdown + debug info (colors enabled)
kubectl broker --endpoint liveness                 # Specific health endpoint with colored status
kubectl broker --statefulset broker --raw          # Unprocessed response (colors disabled)
kubectl broker --pod broker-0 --endpoint readiness # Readiness check with colored indicators
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

### 🚀 Binary Size Optimization (Completed)
- Optimized from 53MB to 35MB (-34% reduction) using selective Kubernetes client imports
- Replaced full `kubernetes.Clientset` with specific typed clients (`CoreV1Client`, `AppsV1Client`)
- Advanced build optimization with `-ldflags="-w -s"`, `-trimpath`, `CGO_ENABLED=0`
- UPX compression support for Linux systems

## Key Technical Decisions

- **Optimized Kubernetes Integration**: Uses specific typed clients (`CoreV1Client`, `AppsV1Client`) instead of full `kubernetes.Clientset` for minimal binary size
- **Programmatic Port-Forwarding**: Self-contained operation using custom REST client for SPDY connections
- **Concurrent Architecture**: Goroutines with dynamic port allocation for parallel health checks
- **Intelligent Defaults**: Context-aware namespace detection and StatefulSet name defaulting
- **User-Centric Installation**: `~/.kubectl-broker/` directory avoiding system-wide installation
- **HiveMQ-Specific**: Targets broker StatefulSets with health endpoints on port named "health"
- **Local Operator Focus**: Designed for diagnostic execution from operator machines
- **Enhanced Health Analysis**: Comprehensive JSON parsing with multiple output formats for both human operators and external tool integration
- **Production Code Quality**: Clean, linter-compliant Go code following best practices with proper error handling

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
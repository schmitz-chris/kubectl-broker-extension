# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a production-ready Go project for `kubectl-broker`, a kubectl plugin CLI tool that streamlines health diagnostics for HiveMQ clusters running on Kubernetes. The project has completed all major phases and is fully functional with intelligent defaults, concurrent health checks, and optimized binary size.

## Project Structure

- `cmd/kubectl-broker/main.go` - Main CLI application with intelligent defaults
- `pkg/` - Core functionality packages (k8s client, port-forwarding, concurrent health checks)
- `PLAN.md` - Implementation roadmap (all phases completed)
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

## HiveMQ Integration

The tool is specifically designed for HiveMQ broker clusters where:
- Brokers run as StatefulSets in Kubernetes
- Each pod exposes a health API endpoint (typically on port 9090)
- Health checks return JSON status information about cluster state, extensions, and MQTT listeners
- Multiple broker instances need to be checked individually to get complete cluster health status
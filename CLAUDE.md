# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a planned Go project for creating `kubectl-broker`, a kubectl plugin CLI tool designed to streamline health diagnostics for HiveMQ clusters running on Kubernetes. The project is currently in the planning phase with documentation files outlining the implementation approach.

## Project Structure

- `PLAN.md` - Detailed 3-phase implementation plan for the kubectl-broker tool
- `OBJECTS.md` - Example Kubernetes objects (StatefulSet, Pod) and health API responses for HiveMQ broker
- `.claude/settings.local.json` - Local permissions configuration for Claude Code

## Development Commands

This is a Go project that will be built using standard Go tooling:

```bash
# Initialize Go module (when ready to start implementation)
go mod init kubectl-broker

# Build the application
go build -o kubectl-broker .

# Get dependencies
go get <package>

# Run tests
go test ./...
```

The CLI tool being built should be executable as:
```bash
# Run with help
./kubectl-broker --help

# Discover broker instances
./kubectl-broker --discover
```

## Architecture

The planned tool follows a phased development approach:

### Phase 1: Single Pod Connection
- Establish core connection logic using native `k8s.io/client-go` library
- Enhanced port discovery with fallback strategy and `--port/-p` flag override
- Implement port-forwarding to query health endpoints directly
- Target single pods specified via `--pod` and `--namespace` flags
- Comprehensive error handling for RBAC issues, pod status, and connectivity problems

### Phase 2: Parallel Cluster Health Checks  
- Scale to discover all pods in a StatefulSet using label selectors
- Dynamic port allocation using `net.Listen(":0")` for concurrent connections
- Implement concurrent health checks across multiple broker instances
- Enhanced error handling for StatefulSet discovery and individual pod failures
- Aggregate and format results in tabular output

### Phase 3: Production Polish
- Add intelligent defaults (namespace from kubectl context, default StatefulSet name "broker")
- Support JSON output format via `--output json`
- Add timeout and verbose logging options
- Comprehensive error handling with actionable guidance for users
- Context-aware error messages that suggest solutions

## Key Technical Decisions

- Uses `k8s.io/client-go` for native Kubernetes API interaction instead of shelling out to kubectl
- Implements port-forwarding programmatically for robust, self-contained operation
- Targets HiveMQ broker StatefulSets with health endpoints on port named "health"
- Designed for local execution by operators running diagnostics from their machines

## HiveMQ Integration

The tool is specifically designed for HiveMQ broker clusters where:
- Brokers run as StatefulSets in Kubernetes
- Each pod exposes a health API endpoint (typically on port 9090)
- Health checks return JSON status information about cluster state, extensions, and MQTT listeners
- Multiple broker instances need to be checked individually to get complete cluster health status
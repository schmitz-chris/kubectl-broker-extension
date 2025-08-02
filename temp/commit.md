# Git Commit Message

```
feat: implement Phase 1 - kubectl-broker single pod health diagnostics

Implement complete Phase 1 functionality for kubectl-broker, a kubectl plugin 
that streamlines health diagnostics for HiveMQ clusters on Kubernetes.

## Key Features Implemented

### Core Functionality
- Single pod health checks via native port-forwarding
- Automatic discovery of health port (named "health") 
- Manual port override with --port/-p flag
- HTTP health check to /api/v1/health endpoint
- Full JSON health response display

### Enhanced Discovery & Usability  
- Discovery mode (--discover) to find broker pods across all namespaces
- Smart kubeconfig detection with kubie support
- Comprehensive error handling with actionable guidance
- Cluster/context debugging information for transparency

### Technical Implementation
- Native k8s.io/client-go integration (no kubectl dependency)
- SPDY-based port-forwarding with lifecycle management
- Dynamic local port allocation using net.Listen(":0")
- Graceful shutdown handling (Ctrl+C support)
- Robust error handling for RBAC, pod status, and connectivity issues

### kubie Integration
- Automatic detection of KUBIE_KUBECONFIG environment variable
- Fallback to KUBECONFIG and default ~/.kube/config
- Full compatibility with kubie context management

## Usage Examples

```bash
# Discover available broker pods
./kubectl-broker --discover

# Check specific broker pod
./kubectl-broker --pod broker-0 --namespace my-namespace

# Manual port specification
./kubectl-broker --pod broker-0 --namespace my-namespace --port 9090
```

## Project Structure
- cmd/kubectl-broker/main.go - CLI application entry point
- pkg/k8s.go - Kubernetes client with kubie support
- pkg/portforward.go - Native port-forwarding implementation
- pkg/discovery.go - Broker discovery across namespaces
- pkg/errors.go - Enhanced error handling with user guidance

Phase 1 delivers a fully functional kubectl plugin that provides robust,
user-friendly health diagnostics for HiveMQ broker clusters with intelligent
defaults and comprehensive error handling.

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```
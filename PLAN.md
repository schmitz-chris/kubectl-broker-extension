Of course. That's the most important goal, and we can easily add an explicit phase to the plan to ensure the final product integrates perfectly as a `kubectl` plugin.

The core logic in Phases 1-3 remains the same, as it defines what the tool *does*. This new final phase will define how it's **packaged and delivered**.

Here is the updated plan with the added integration phase.

***

### **Rationale**

This plan outlines the creation of `kubectl-broker`, a CLI tool designed to streamline health diagnostics for HiveMQ clusters on Kubernetes. The primary goal is to provide a single command that automates the complex process of checking the health status of every node in a cluster.

The key technical decision is to implement the connection logic using the native **`k8s.io/client-go`** library instead of shelling out to the `kubectl` binary. This approach creates a **robust, self-contained application** with no external dependencies, leading to a more reliable user experience and simpler distribution. The plan prioritizes local execution via automated port-forwarding, as this is the primary use case for operators running diagnostics from their own machines. It progresses from a single-node connection to a full, parallelized cluster check, culminating in a polished tool with intelligent, context-aware defaults that is packaged as a standard `kubectl` plugin.

***

### ## Phase 1: Foundation - Single Pod Native Connection ✅ **COMPLETED**

**Goal:** Establish and validate the core connection logic by creating a native port-forward to a single, specified pod and querying its health endpoint.

**Implementation Details:**
* **Project Structure**: Go module with `cmd/kubectl-broker/` and `pkg/` directories
* **CLI Framework**: Cobra command with three flags:
  - `--pod` (required): Name of the pod to check
  - `--namespace/-n` (required): Namespace of the pod  
  - `--port/-p` (optional): Manual port override for health check
* **Discovery Mode**: Added `--discover` flag to find available broker pods across all accessible namespaces
* **kubie Integration**: Smart kubeconfig detection that checks for:
  - `KUBIE_KUBECONFIG` environment variable (kubie-specific)
  - `KUBECONFIG` environment variable (standard)
  - Default `~/.kube/config` as fallback
* **Enhanced Port Discovery**: Automatic discovery of port named **`"health"`** with intelligent fallback:
  - Primary: Search for container port with name "health" 
  - Fallback: Display helpful error message listing all available ports with names and numbers
  - Manual Override: `--port/-p` flag allows direct port specification
  - Example error: "Health port not found. Available ports: [mqtt(8883), health(9090), metrics(9399)]. Use --port/-p to specify manually."
* **Native Port-Forward**: Uses **`k8s.io/client-go/tools/portforward`** library with:
  - SPDY dialer for robust connections
  - Dynamic local port allocation using `net.Listen(":0")` 
  - Proper lifecycle management with ready/stop channels
  - Signal handling for graceful shutdown (Ctrl+C)
* **Comprehensive Error Handling**: User-friendly messages for common scenarios:
  - RBAC/Permission: "Access denied. Ensure your kubeconfig has permissions to list/portforward pods in namespace 'X'"
  - Pod status validation: "Pod 'broker-0' is not ready (status: Pending/CrashLoopBackOff). Cannot establish connection."
  - Network connectivity: "Failed to establish port-forward to pod 'broker-0'. Check if pod is running and network policies allow connection."
  - Context issues: "No current context set in kubeconfig" with guidance
* **Health Check**: HTTP GET request to `/api/v1/health` endpoint with full JSON response display
* **Debugging Information**: Shows current cluster, server, context, and kubeconfig source for transparency

**Delivered Features:**
- ✅ Single pod health checks with automatic port discovery
- ✅ Manual port override capability  
- ✅ kubie context manager full compatibility
- ✅ Discovery mode to find broker pods across clusters
- ✅ Comprehensive error handling with actionable guidance
- ✅ Native Kubernetes integration without external dependencies

***

### ## Phase 2: Scaling Up - Parallel Connections and Aggregation ✅ **COMPLETED**

**Goal:** Extend the single-pod logic to discover all pods in a StatefulSet and perform health checks concurrently, presenting the results in an aggregated view.

**Implementation Details:**
* **Dual Mode CLI Interface**: Added `--statefulset` flag for cluster mode while preserving `--pod` flag for single pod mode:
  - Smart flag validation prevents using both `--statefulset` and `--pod` together
  - Maintains backward compatibility with Phase 1 functionality
  - Enhanced help text clearly distinguishes between single pod and cluster modes
* **StatefulSet Discovery**: Native Kubernetes integration for pod discovery:
  - `GetStatefulSet()` method retrieves StatefulSet metadata and validates existence
  - `GetPodsFromStatefulSet()` uses label selectors to find all pods belonging to the StatefulSet
  - Automatic filtering based on StatefulSet's `.spec.selector.matchLabels`
* **Concurrent Health Checking Architecture**: 
  - **Goroutine-based Parallelism**: One goroutine per pod using `sync.WaitGroup` for coordination
  - **Dynamic Port Allocation**: Each pod gets unique local port via `net.Listen(":0")` to prevent conflicts
  - **Timeout Management**: 30-second context timeout per pod prevents hanging on unresponsive instances
  - **Silent Mode**: Quiet health checks for cluster mode (no verbose per-pod output)
* **Professional Tabular Output**: 
  - Clean table formatting using `text/tabwriter` with consistent column alignment
  - Displays: Pod Name | Status | Health Port | Local Port | Response Time | Details
  - Summary statistics showing healthy vs total pods with emoji indicators
  - Truncated details for long error messages (50 char limit with "...")
* **Enhanced Error Handling**: Comprehensive error categories and graceful degradation:
  - **StatefulSet-specific errors**: "StatefulSet 'X' not found. Use --discover to find available StatefulSets"
  - **Individual pod resilience**: Failed pods don't stop cluster-wide health checks
  - **Status categorization**: HEALTHY, POD_NOT_READY, PORT_DISCOVERY_FAILED, HEALTH_CHECK_FAILED, etc.
  - **Response time tracking**: Measures actual health check duration for performance insights
* **Updated Discovery Mode**: Enhanced `--discover` output shows both single and cluster commands:
  - Single pod: `./kubectl-broker --pod broker-0 --namespace X`
  - All pods: `./kubectl-broker --statefulset broker --namespace X`

**Delivered Features:**
- ✅ Concurrent health checks for entire StatefulSets (2-pod and 3-pod clusters tested)
- ✅ Dynamic port allocation preventing port conflicts during parallel checks
- ✅ Professional tabular output with response times and status categorization
- ✅ Graceful error handling - individual pod failures don't stop entire operation
- ✅ Backward compatibility - Phase 1 single pod functionality fully preserved
- ✅ Enhanced discovery mode showing both single and cluster usage examples
- ✅ StatefulSet-specific error messages with actionable guidance

***

### ## Phase 3: Polish - Intelligent Defaults and Usability

**Goal:** Make the tool effortless for common use cases by implementing intelligent defaults and adding production-ready features for robustness and automation.

**Key Steps:**
* **Intelligent Defaults**: Make flags optional with smart fallbacks:
  - Namespace defaults to the **current `kubectl` context**
  - StatefulSet defaults to **`"broker"`**
  - Context handling: "No current kubectl context set. Use 'kubectl config use-context' or specify --namespace"
* **Production Features**: Add `--output json`, `--timeout`, and `--verbose` flags
* **Comprehensive Error Handling**: Provide actionable guidance for all common scenarios:
  - RBAC issues with specific permission requirements
  - Missing resources with suggestions for available alternatives  
  - Network connectivity problems with troubleshooting steps
  - Configuration issues with clear resolution paths
* **User Experience**: Ensure error messages guide users to solutions rather than just reporting problems.

***

### ## Phase 4: Integration - Packaging as a `kubectl` Plugin ✅ **COMPLETED**

**Goal:** Ensure the compiled application is correctly built and installed to function seamlessly as a `kubectl` plugin.

**Implementation Details:**
* **Executable Naming Convention**: 
  - Binary correctly named `kubectl-broker` for kubectl plugin discovery
  - Go build process configured: `go build -o kubectl-broker ./cmd/kubectl-broker`
  - Executable permissions properly set with chmod +x
* **User-Friendly Installation System**: 
  - **`~/.kubectl-broker/` Directory**: Clean, user-specific installation location avoiding system directories
  - **Automated Installation Script**: `install.sh` with intelligent shell detection (bash/zsh)
  - **PATH Management**: Automatic addition to shell RC files (.bashrc, .zshrc, .bash_profile)
  - **Cross-Platform Support**: macOS and Linux shell detection and configuration
* **Professional Build System**:
  - **Comprehensive Makefile**: Targets for build, install, clean, uninstall, test, and cross-compilation
  - **Development Builds**: Race detector support with `make dev`
  - **Release Builds**: Optimized binaries with `make release`
  - **Cross-Compilation**: Multi-platform binaries (Linux, macOS, Windows) for distribution
* **Plugin Integration Verification**:
  - ✅ kubectl plugin discovery working: `kubectl plugin list | grep broker`
  - ✅ Plugin invocation working: `kubectl broker --help`
  - ✅ Full functionality through kubectl: `kubectl broker --discover`, `kubectl broker --statefulset`
  - ✅ Discovery output correctly shows `kubectl broker` commands instead of direct binary calls
* **Documentation and User Experience**:
  - **Comprehensive README**: Installation instructions, usage examples, troubleshooting
  - **Installation Verification**: Step-by-step testing instructions for users
  - **Plugin Usage Examples**: Complete documentation for kubectl broker usage patterns

**Delivered Features:**
- ✅ kubectl plugin discovery and invocation working seamlessly
- ✅ User-friendly installation in `~/.kubectl-broker/` directory  
- ✅ Automated installation script with PATH management
- ✅ Professional Makefile with comprehensive build targets
- ✅ Cross-platform build support (Linux, macOS, Windows)
- ✅ Discovery output shows correct kubectl plugin commands
- ✅ Complete installation and usage documentation
- ✅ Verified functionality across all plugin features

**Final Plugin Commands:**
```bash
# Installation
make install-auto

# Usage as kubectl plugin
kubectl broker --discover
kubectl broker --pod broker-0 --namespace my-namespace
kubectl broker --statefulset broker --namespace my-namespace
```
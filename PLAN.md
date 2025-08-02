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

### ## Phase 2: Scaling Up - Parallel Connections and Aggregation

**Goal:** Extend the single-pod logic to discover all pods in a StatefulSet and perform health checks concurrently, presenting the results in an aggregated view.

**Key Steps:**
* Change the command to use `--statefulset` and `--namespace` flags.
* Discover all pods belonging to the StatefulSet using label selectors.
* **Dynamic Port Allocation**: For each pod, launch a goroutine to manage its own unique, concurrent port-forward session:
  - Use `net.Listen(":0")` to get a random available local port from the OS
  - This ensures each goroutine gets a unique local port, preventing conflicts when checking multiple pods simultaneously
  - Each port-forward connection uses the **`k8s.io/client-go/tools/portforward`** library
* **Enhanced Error Handling**: Handle pod-specific issues gracefully:
  - StatefulSet not found: "StatefulSet 'broker' not found in namespace 'default'. Available StatefulSets: [list them]"
  - Namespace issues: "Namespace 'X' not found or inaccessible"
  - Individual pod failures: Continue with other pods, report which ones failed and why
* Aggregate the results from all goroutines and present them in a formatted table using **`text/tabwriter`**.

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

### ## Phase 4: Integration - Packaging as a `kubectl` Plugin

**Goal:** Ensure the compiled application is correctly built and installed to function seamlessly as a `kubectl` plugin.

1.  **Executable Naming Convention:**
    * The Go build process must produce an executable file named exactly **`kubectl-broker`**. This is the fundamental requirement for `kubectl` to discover it as a plugin for the `broker` command. This will be configured in the build script (e.g., `go build -o kubectl-broker ./cmd/kubectl-broker`).

2.  **Installation Path:**
    * The plan must include documenting clear installation instructions for users. These instructions will direct users to place the **`kubectl-broker`** executable file into a directory that is included in their system's **`$PATH`** environment variable.

3.  **Command Invocation:**
    * The `cobra` command structure we've designed naturally supports the plugin model. When a user runs `kubectl broker --statefulset my-cluster`, `kubectl` will find and execute `kubectl-broker`, passing the `--statefulset my-cluster` arguments directly to it. No changes to the command logic are needed for this to work.
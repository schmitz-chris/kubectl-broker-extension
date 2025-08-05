# PLAN.md - Enhanced HiveMQ Health API Analysis

## Project Overview

This is a production-ready Go project for `kubectl-broker`, a kubectl plugin CLI tool that streamlines health diagnostics for HiveMQ clusters running on Kubernetes. All major phases (1-4) have been completed successfully, and the tool is fully functional with intelligent defaults, concurrent health checks, and optimized binary size.

## Completed Phases ✅

### Phase 1: Foundation - Single Pod Native Connection ✅ COMPLETED
- Native Kubernetes integration with `k8s.io/client-go`
- Single pod health checks with automatic port discovery
- Discovery mode to find broker pods across clusters
- Comprehensive error handling with actionable guidance

### Phase 2: Scaling Up - Parallel Connections and Aggregation ✅ COMPLETED  
- Concurrent health checks for entire StatefulSets
- Dynamic port allocation preventing conflicts during parallel checks
- Professional tabular output with response times and status categorization
- Graceful error handling - individual pod failures don't stop entire operation

### Phase 3: Polish - Intelligent Defaults and Usability ✅ COMPLETED
- Intelligent defaults for StatefulSet and namespace selection
- Context-aware namespace detection from kubectl configuration
- Visual feedback showing which defaults were applied
- Enhanced error handling for context configuration issues

### Phase 4: Integration - Packaging as kubectl Plugin ✅ COMPLETED
- kubectl plugin discovery and invocation working seamlessly
- User-friendly installation in `~/.kubectl-broker/` directory
- Automated installation script with PATH management
- Professional Makefile with comprehensive build targets
- Binary size optimization (53MB → 35MB, -34% reduction)

---

## Phase 5: Enhanced Health API Analysis ✅ COMPLETED

### Implementation Completed
- ✅ Parse and analyze HiveMQ health JSON responses for detailed diagnostics
- ✅ Create Go structs for HiveMQ health JSON structure in `pkg/health/`
- ✅ Parse `status` field (UP, DOWN, DEGRADED, UNKNOWN, OUT_OF_SERVICE)
- ✅ Extract `components` section for detailed component health
- ✅ Parse `details` section for additional diagnostic information

### Enhanced Output Formats ✅ COMPLETED
- ✅ Add `--json` flag for raw JSON output (machine-parseable for external tools)
- ✅ Add `--raw` flag for unprocessed health endpoint responses
- ✅ Enhanced table format showing parsed health status instead of "HEALTHY"
- ✅ Color-coded status indicators: [UP], [DOWN], [DEGRADED], [UNKNOWN]
- ✅ Component-specific details in detailed mode

### Multiple Health Endpoints ✅ COMPLETED
- ✅ System health `/api/v1/health/` (full health information)
- ✅ Liveness check `/api/v1/health/liveness` (basic availability)
- ✅ Readiness check `/api/v1/health/readiness` (ready to serve traffic)
- ✅ Add `--endpoint` flag to specify which health endpoint to query

### Advanced Diagnostics Mode ✅ COMPLETED
- ✅ Add `--detailed` flag for expanded component breakdown
- ✅ Enhanced error reporting with actionable guidance based on health status
- ✅ Component-specific health details (Cluster, MQTT, Extensions)
- ✅ Support for multiple output formats (table, json, raw)
- ✅ Color-coded health status indicators for improved visual recognition

---

## Phase 6: Subcommand Architecture ✅ COMPLETED

### Goal
Transform the single-purpose health checker into a comprehensive HiveMQ cluster management tool with multiple commands and extensible architecture.

### Current Limitations
- Tool serves only one purpose (health checking)
- No structure for additional HiveMQ management features
- Limited extensibility for future cluster operations

### Phase 6.1: Command Restructuring ✅ COMPLETED
**Goal:** Implement subcommand-based CLI architecture

**Implementation:**
- ✅ Restructure main.go to use parent command without direct functionality
- ✅ Create `status` subcommand containing current health check functionality
- ✅ Create `backup` subcommand framework (placeholder implementation)
- ✅ Root command shows available subcommands when called without arguments

### Phase 6.2: Command Separation ✅ COMPLETED
**Goal:** Clean separation of concerns between different tool functions

**Implementation:**
- ✅ `cmd/kubectl-broker/main.go` - Root command and subcommand registration
- ✅ `cmd/kubectl-broker/status.go` - Status/health checking functionality
- ✅ `cmd/kubectl-broker/backup.go` - Backup operations (framework only)
- ✅ Maintain all existing flags and functionality for status command

### Usage Examples
```bash
# Show available commands
kubectl broker

# Health status checking (current functionality)
kubectl broker status
kubectl broker status --statefulset broker --namespace production --detailed
kubectl broker status --json --endpoint liveness

# Future backup functionality
kubectl broker backup --statefulset broker --namespace production
kubectl broker backup --output-dir ./backups
```

### Future Command Roadmap
- `kubectl broker backup` - Create HiveMQ cluster backups
- `kubectl broker restore` - Restore from backup archives
- `kubectl broker maintenance` - Cluster maintenance operations
- `kubectl broker config` - Configuration management
- `kubectl broker logs` - Enhanced log collection and analysis

### Benefits
- **Extensibility:** Easy addition of new management features
- **Organization:** Clear separation between different tool functions
- **User Experience:** Intuitive command structure following kubectl patterns
- **Maintenance:** Easier code organization and testing per command

---

## Project Structure
```
kubectl-broker/
├── cmd/kubectl-broker/    # Main CLI application with subcommands
│   ├── main.go           # Root command and subcommand registration
│   ├── status.go         # Status/health checking subcommand (Phase 6)
│   └── backup.go         # Backup operations subcommand (Phase 6)
├── pkg/                   # Core functionality packages  
│   ├── concurrent.go      # Parallel health checking logic
│   ├── discovery.go       # Pod/StatefulSet discovery
│   ├── errors.go          # Enhanced error handling
│   ├── health/            # HiveMQ health response parsing (Phase 5)
│   ├── k8s.go            # Kubernetes client wrapper
│   └── portforward.go     # Port forwarding implementation
├── install.sh            # Automated installation script
├── Makefile              # Professional build system
├── README.md             # Complete user documentation
├── PLAN.md               # Implementation roadmap (this file)
└── OBJECTS.md            # HiveMQ Kubernetes object examples
```

## Current Status
- **Phases 1-6:** Production ready ✅
- **Future Phases:** Available for additional HiveMQ management features
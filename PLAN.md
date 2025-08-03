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

## New Phase 5: Enhanced Health API Analysis 🚧 IN PROGRESS

### Current Limitations
- Simple HTTP GET to `/api/v1/health` endpoint only
- Only checks HTTP 200 status, no JSON parsing or analysis
- Basic "HEALTHY/FAILED" output without diagnostic details
- Missing rich health information that HiveMQ Health API provides

### Phase 5.1: Health Response Parsing
**Goal:** Parse and analyze HiveMQ health JSON responses for detailed diagnostics

**Implementation:**
- Create Go structs for HiveMQ health JSON structure
- Parse `status` field (UP, DOWN, DEGRADED, UNKNOWN, OUT_OF_SERVICE)
- Extract `components` section for detailed component health
- Parse `details` section for additional diagnostic information

### Phase 5.2: Enhanced Output Formats
**Goal:** Provide multiple output formats for different use cases

**Implementation:**
- Add `--json` flag for raw JSON output (machine-parseable for external tools)
- Add `--raw` flag for unprocessed health endpoint responses
- Enhanced table format showing parsed health status instead of "HEALTHY"
- Status indicators: [UP], [DOWN], [DEGRADED], [UNKNOWN] (no emojis)
- Component-specific details in detailed mode

### Phase 5.3: Multiple Health Endpoints
**Goal:** Support different HiveMQ health endpoints for specific diagnostics

**Implementation:**
- System health `/api/v1/health/` (current implementation)
- Liveness check `/api/v1/health/liveness` (basic availability)
- Readiness check `/api/v1/health/readiness` (ready to serve traffic)
- Add `--endpoint` flag to specify which health endpoint to query

### Phase 5.4: Advanced Diagnostics Mode
**Goal:** Provide detailed component analysis and enhanced error reporting

**Implementation:**
- Add `--detailed` flag for expanded component breakdown
- Enhanced error reporting with actionable guidance based on health status
- Component-specific health details (Cluster, MQTT, Extensions)
- Support for multiple output formats (table, json, raw)

### Implementation Strategy
- Extend `HealthCheckResult` struct with parsed health data and raw JSON
- Create new `pkg/health` package for HiveMQ-specific health response parsing
- Add CLI flags: `--json`, `--raw`, `--detailed`, `--endpoint`
- Maintain backward compatibility with current simple health checks
- Update concurrent health check functions to support new analysis

### Expected Usage Examples
```bash
# Current simple usage (unchanged)
kubectl broker

# Raw JSON output for external parsing
kubectl broker --json

# Detailed component analysis
kubectl broker --detailed

# Specific health endpoint
kubectl broker --endpoint liveness

# Combined options
kubectl broker --json --endpoint readiness --detailed
```

### Benefits
- **Human Analysis:** Rich diagnostic information with component breakdown
- **Machine Integration:** JSON output for jq, monitoring tools, scripts
- **Flexible Diagnostics:** Different health endpoints for specific use cases
- **Backward Compatibility:** Existing usage patterns continue to work
- **External Tool Support:** Raw JSON enables integration with monitoring pipelines

---

## Project Structure
```
kubectl-broker/
├── cmd/kubectl-broker/    # Main CLI application
├── pkg/                   # Core functionality packages  
│   ├── concurrent.go      # Parallel health checking logic
│   ├── discovery.go       # Pod/StatefulSet discovery
│   ├── errors.go          # Enhanced error handling
│   ├── health/            # NEW: HiveMQ health response parsing
│   ├── k8s.go            # Kubernetes client wrapper
│   └── portforward.go     # Port forwarding implementation
├── install.sh            # Automated installation script
├── Makefile              # Professional build system
├── README.md             # Complete user documentation
├── PLAN.md               # Implementation roadmap (this file)
└── OBJECTS.md            # HiveMQ Kubernetes object examples
```

## Current Status
- **Phases 1-4:** Production ready ✅
- **Phase 5:** Enhanced health analysis - In development 🚧
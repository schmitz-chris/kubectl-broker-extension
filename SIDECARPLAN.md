# Sidecar Integration Plan

## 1. Background & Goals
- Extend `kubectl broker backup` so operators can use the existing HiveMQ management API flow **and** the new backup sidecar (`schmitz-chris/broker-backup`).
- Keep current UX standards (namespace resolution via `cli_helpers`, `--output` handling, color rules) while exposing sidecar-specific capabilities (S3 listing, dry-run restore, Token Vendor validation).
- Maintain feature parity for: create, list, download, status, restore; add sidecar-only subfeatures without breaking existing scripts.

## 2. Scope Overview
1. **Client Layer** – Add a dedicated Go client for the sidecar REST API plus wrappers for Kubernetes port-forwarding.
2. **CLI Wiring** – Introduce flags and modes so each `backup` subcommand can target either the management API or the sidecar.
3. **Output & UX** – Provide consistent table/JSON/YAML rendering for new responses, reuse helper patterns, document workflows.
4. **Quality Guardrails** – gofmt, `go test ./...`, smoke instructions for local Token Vendor + sidecar testbed.

## 3. External Dependencies & Assumptions
- Sidecar exposes HTTP endpoints documented in `broker-backup` (default port `RESTORE_PORT=8085`).
- Token Vendor is reachable from the sidecar and handles S3 credentials; CLI never talks to AWS directly.
- Namespace → UUID mapping lives inside the sidecar/token vendor stack; CLI only needs the Kubernetes namespace/pod.

## 4. Detailed Implementation Steps

### Phase A – Sidecar Client Package
1. Create `pkg/sidecar` with:
   - `Client` struct (`NewClient(baseURL string)`), auth headers if required later.
   - Methods: `ListLocalBackups`, `ListRemoteBackups(limit int)`, `Restore(RemoteRestoreRequest)`, `Purge(PurgeRequest)`, `Upload`, `Metrics()` (optional streaming), mirroring documented endpoints.
2. Add `Options` similar to `backup.BackupOptions` (namespace, pod override, username/password if the sidecar adds auth later, timeout, poll interval, output dir, dry run flag).
3. Implement Kubernetes orchestration helpers:
   - `ResolveSidecarPod(ctx, k8sClient, namespace, statefulSet, podFlag)` chooses a pod (pod flag > sidecar label > first StatefulSet pod) and surfaces errors via `pkg.EnhanceError`.
   - `WithPodPortForward(ctx, podName, namespace, remotePort, func(baseURL string) error)` reuses existing port-forward utility for pods (not services).
4. Define response structs that match the JSON payloads from `broker-backup` docs (`list/local`, `list-remote`, restore responses, metrics). Include validation helpers.

### Phase B – CLI Flag & Mode Updates (`cmd/kubectl-broker/backup.go`)
1. Auto-detect whether the sidecar is reachable; prefer it when available and fall back to the management API without requiring a global flag.
2. Add `--pod` (existing list command already references pods? confirm; if not, introduce) and `--sidecar-port` (default 8085) so users can aim at a specific broker pod/sidecar.
3. Subcommand-specific flags:
   - `list`: Always call `ListRemoteBackups` (S3 inventory) with an optional `--limit` parameter.
   - `download`: Reserve additional flags for future sidecar-aware transfers (no `--remote` toggle).
   - `restore`: `--source {local,remote}`, `--version`, `--dry-run`; default `local` for management API parity.
4. Ensure flag conflict checks follow the shared helpers (e.g., `checkMutuallyExclusiveFlags`). Errors must follow the “Please either:” guidance.

### Phase C – Command Execution Paths
1. Refactor each subcommand so the list path always hits the sidecar’s remote inventory, while management-only paths keep their existing behavior.
2. Implement sidecar variants using the new client package. Responsibilities:
   - Resolve namespace/statefulset defaults (existing `applyBackupDefaults`).
   - Determine pod + remote port; use pod port-forwarding to build `http://localhost:<port>`.
   - Parse/format responses with new output helpers described below.
3. Preserve existing behavior for management engine to avoid regressions.

### Phase D – Output & Presentation Helpers
1. Create `cmd/kubectl-broker/backup_output.go` (if not already) or extend it to include:
   - Table schema and renderers for sidecar remote lists (columns: `OBJECT`, `SIZE`, `AGE`).
   - JSON/YAML emitters using `currentOutputFormat()`; tables only in `table` mode.
2. Add dedicated formatting helpers for restore responses (show mode, version, dry-run indicator, status message). Use `colorOutputEnabled()` for status text.
3. Update README / CLI help: describe new flags, mention Token Vendor + sidecar requirements, add examples for remote restore/dry run.

### Phase E – Validation & Testing
1. Run `gofmt` on all new/changed Go files; ensure import ordering compliance (stdlib, third-party, internal).
2. Create at least one unit test file in `pkg/sidecar` (e.g., response parsing and filename derivation). Consider interface-based tests with fake HTTP servers.
3. Execute `go test ./...` locally. Document any sandbox blockers; rerun with approval if necessary.
4. Provide a local smoke-test recipe in `SIDECARPLAN.md` appendix or README, e.g.:
   - `gh repo clone schmitz-chris/broker-backup`
   - `make build-sidecar`
   - Run Token Vendor + sidecar locally (per docs) and call new CLI flags using `kubectl broker backup restore --source remote --version ... --dry-run`.

## 5. Open Questions / Follow-Ups
- Does the sidecar require auth headers or is it network-limited? Add flags later if auth is introduced.
- Should `kubectl broker backup create` trigger the sidecar watcher (e.g., `/v1/backup/upload`) or stay HiveMQ-only? Current plan keeps create in management API mode until requirements change.
- How should errors from Token Vendor (HTTP 4xx) be surfaced? Likely rewrapped via `pkg.EnhanceError` with actionable guidance.

## 6. Timeline Estimate
1. Phase A: 1 day (client + helpers + tests).
2. Phase B/C: 1–1.5 days (CLI wiring + engine switching logic).
3. Phase D/E & docs: 0.5 day.
Total: ~3 workdays including manual smoke testing.

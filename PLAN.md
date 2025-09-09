
Incremental Implementation Plan: Single-binary kubectl-broker and kubectl-pulse (macOS)

Rationale
- Maximize code reuse and reduce maintenance by delivering both kubectl broker and kubectl pulse from a single binary whose behavior depends on the invocation name. This aligns with established patterns and leverages the existing pkg/ infrastructure for Kubernetes integration, health checks, and shared output/error handling [1].
- Simplify distribution and installation by using a symlink-based approach so kubectl discovers two plugins from one binary, reducing build and release complexity [1].
- Maintain a consistent user experience across both products by standardizing flags, output formats, and error handling, while keeping product-specific commands separate and context-aware [1].
- Address CI and non-interactive usage by introducing robust color control and machine-readable outputs (json/yaml) without adding automated test coverage in this iteration to ship fast.
- Rely on the existing architecture benefits (optimized Kubernetes client, concurrent health checks, professional error handling, color-coded displays, intelligent defaults) without duplicating logic [1].

Scope and non-goals
- In scope: macOS only; symlink-based dual-plugin from a single binary; dynamic help/UX by invoked name; global color/output behavior; manual validation.
- Out of scope: Windows/Linux support, installer packaging for additional platforms, automated tests, and major new subcommands.

Deliverables
- A single binary that serves kubectl broker and kubectl pulse based on invocation name, with separate command trees and context-aware help [1].
- Updated install targets that create a symlink for kubectl-pulse pointing to kubectl-broker, discoverable by kubectl plugin list [1].
- Global output and color controls: --no-color and --output table|json|yaml, with environment-based and TTY-aware color behavior.
- Documentation updates for usage, installation, and behavior.

Phase 1: Command structure and routing
Objective: Route behavior based on invoked name and integrate Pulse commands into the main command router.
- Mode detection
    - Detect effective invocation name to set product mode: broker vs pulse.
    - Define an internal product context variable available to command initialization and help text generation.
- Command routers
    - Integrate existing Pulse command definitions from cmd/kubectl-broker/pulse.go into the primary command router, ensuring shared pkg/ infrastructure is used for Kubernetes connectivity and concurrent diagnostics [1].
    - Ensure product scoping: broker-only commands are hidden/disabled in pulse mode and vice versa. The shared global flags and error/formatting are identical across modes [1].
- Shared pkg/ integration
    - Confirm all Kubernetes interactions go through pkg.NewK8sClient() and health diagnostics go through pkg.PerformConcurrentHealthChecks(), avoiding duplication [1].
    - Ensure centralized error handling and output formatting paths are used by both product trees [1].
- UX consistency
    - Normalized global flags (namespace, kubeconfig, context, etc.) applied uniformly to both modes.
    - Ensure exit codes and error semantics remain consistent across modes.

Phase 2: Installation system updates (macOS)
Objective: Install one binary and create a symlink so kubectl discovers two plugins.
- Installer behavior
    - Copy the kubectl-broker binary to a directory on PATH (macOS default: /usr/local/bin or equivalent user-local bin).
    - Create a symlink named kubectl-pulse that points to the kubectl-broker binary in the same directory [1].
    - Verify kubectl plugin discovery: both kubectl-broker and kubectl-pulse should be listed under kubectl plugin list [1].
- Build and make targets
    - Provide a make target to build the single binary and an install target that performs copy + symlink creation [1].
- Readme note
    - Document that PATH must include the installation directory for kubectl plugin discovery.

Phase 3: Output formats and color behavior
Objective: Make output safe for CI and non-TTY, and provide machine-readable formats.
- Global flags
    - Add --no-color to force-disable ANSI color output at the top-level command (applies to all subcommands).
    - Add --output with allowed values: table (default), json, yaml. The flag is global and applies to relevant commands in both modes.
- Environment and TTY handling
    - Honor NO_COLOR to disable colors if set.
    - Honor CLICOLOR_FORCE=1 to force-enable colors unless overridden by --no-color.
    - Treat CI being set in the environment as a signal to disable colors by default.
    - Detect TTY for stdout; disable colors when stdout is not a TTY.
    - Precedence order: --no-color > NO_COLOR > CLICOLOR_FORCE > CI > TTY detection.
- Rendering pipeline
    - Centralize color enablement decision in the shared formatting layer so all commands uniformly obey color settings.
    - Provide a unified printer abstraction capable of producing:
        - Table (human-optimized; respects color setting).
        - JSON (stable schema appropriate for status/health results).
        - YAML (same schema as JSON).
    - Ensure error and informational messages respect color settings and do not emit ANSI codes when disabled.
- Defaults and backward compatibility
    - Default to table output for interactive users.
    - Default to no color in non-TTY and CI contexts without user intervention.
    - Maintain current human-readable formatting when --output is not specified.

Phase 4: Documentation and help systems
Objective: Make help context-aware and update public docs for dual-plugin usage.
- Context-aware help
    - Adjust root usage, descriptions, and examples based on product mode (invoked name), so kubectl pulse --help is Pulse-focused and hides Broker-specific commands [1].
- Separate documentation sections
    - Maintain separate product sections in README or docs for Broker vs Pulse operations and examples [1].
    - Include examples of --no-color, NO_COLOR, and --output json/yaml usage and behavior.
- Version and metadata
    - Ensure version output is identical across modes except for product naming in help/usage where applicable.

Phase 5: Manual validation (no automated tests for this iteration)
Objective: Validate functionality quickly on macOS.
- Discovery and basic function
    - Confirm kubectl plugin list shows both plugins after installation [1].
    - Run kubectl broker --help and kubectl pulse --help to verify context-aware help.
- Color behavior
    - Verify color is enabled in an interactive terminal by default.
    - Verify color is disabled when piped (non-TTY), when CI is set, when NO_COLOR is set, and when --no-color is specified.
    - Verify CLICOLOR_FORCE=1 enables color unless --no-color is also set.
- Output formats
    - Validate table/json/yaml outputs for key commands (e.g., status/health checks) with consistent schema and content across broker/pulse where shared.
- Consistency
    - Compare shared command behavior and outputs between broker and pulse modes to ensure parity.
- Error handling
    - Confirm errors are formatted consistently and exit codes are non-zero on failure.

Phase 6: Release and distribution notes (macOS)
Objective: Publish artifacts and communicate changes.
- Release artifacts
    - Provide a single binary in tarball or package with instructions to create the symlink during installation [1].
- Changelog
    - Document the single-binary dual-plugin change, color/output behaviors, and macOS-only scope for this release.

Acceptance criteria
- Two kubectl plugins, kubectl broker and kubectl pulse, both functional and discoverable via symlink from a single binary [1].
- Single codebase with no duplication of Kubernetes integration logic; both modes use shared pkg/ functionality [1].
- Consistent UX: identical global flags, output formats, and error handling across both modes [1].
- Color handling follows precedence rules and is disabled by default in non-TTY and CI environments; users can force-disable with --no-color.
- Output supports table (default), json, and yaml with stable schemas for shared status/health commands.
- Documentation and help are context-aware and updated for dual-plugin usage [1].
- Manual validation checklist passes on macOS; no automated test coverage required for this iteration.

Risks and mitigations
- Symlink not on PATH: Mitigate by documenting install location and PATH requirements; validate with kubectl plugin list.
- Color detection edge cases: Provide explicit --no-color and document NO_COLOR, CI, and CLICOLOR_FORCE behavior.
- Schema drift between modes: Centralize printers and data models in shared pkg/ to ensure parity.
- Future platform support: Keep install logic abstracted so Windows/Linux can adopt copy/hardlink/shim strategies later.

Future work (post-iteration)
- Cross-platform installers and discovery (Linux/Windows variants).
- Automated tests: E2E plugin discovery, parity tests for outputs, and CI matrices.
- Packaging for Homebrew and other channels to create the symlink automatically.
- RBAC and persona-specific documentation if permissions differ between broker and pulse.
- Schema versioning and machine-readable output guarantees for long-term stability.

References
- Symlink-based dual-plugin approach, command integration into shared infrastructure, install steps, documentation updates, and expected outcomes derive from the kubectl-broker Extension Implementation Plan [1].

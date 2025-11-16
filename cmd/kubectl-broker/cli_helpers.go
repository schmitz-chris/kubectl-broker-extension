package main

import (
	"fmt"

	"kubectl-broker/pkg"
)

const namespaceGuidanceBase = `failed to determine default namespace: %w

Please either:
- Set a kubectl context with namespace: kubectl config set-context --current --namespace=<namespace>
- Specify namespace explicitly: --namespace <namespace>`

// resolveNamespace returns the provided namespace or falls back to the current kubectl context.
// The second return value indicates whether the namespace came from the context.
func resolveNamespace(value string, includeAllHint bool) (string, bool, error) {
	if value != "" {
		return value, false, nil
	}

	namespace, err := pkg.GetDefaultNamespace()
	if err != nil {
		return "", false, namespaceResolutionError(err, includeAllHint)
	}

	return namespace, true, nil
}

func namespaceResolutionError(err error, includeAllHint bool) error {
	message := namespaceGuidanceBase
	if includeAllHint {
		message += "\n- Use --all-namespaces for cluster-wide operations"
	}
	return fmt.Errorf(message, err)
}

// applyDefaultStatefulSet ensures we always target a StatefulSet when no explicit value was supplied.
// Returns true if the default was applied.
func applyDefaultStatefulSet(value string) (string, bool) {
	if value != "" {
		return value, false
	}
	return "broker", true
}

// mutuallyExclusive ensures that only one of the provided flags is active at the same time.
func mutuallyExclusive(flagA bool, nameA string, flagB bool, nameB string) error {
	if flagA && flagB {
		return fmt.Errorf("cannot use both %s and %s flags together", nameA, nameB)
	}
	return nil
}

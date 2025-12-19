package workflow

// MergeAll returns an aggregator that merges all keys from all branch states.
// This is the default behavior when no aggregator is provided to NewParallel.
func MergeAll() Aggregator {
	return func(state *State, results map[string]*StepResult, errors map[string]error) error {
		for _, result := range results {
			if branchState, ok := GetBranchState(result); ok {
				state.Merge(branchState)
			}
		}
		return nil
	}
}

// MergeKeys returns an aggregator that merges only the specified keys from branch states.
// Keys not in the list are ignored. Later branches overwrite earlier ones for the same key.
func MergeKeys(keys ...string) Aggregator {
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}

	return func(state *State, results map[string]*StepResult, errors map[string]error) error {
		for _, result := range results {
			if branchState, ok := GetBranchState(result); ok {
				for _, k := range keys {
					if val, exists := branchState.Get(k); exists {
						state.Set(k, val)
					}
				}
			}
		}
		return nil
	}
}

// MergeTypedKey returns an aggregator that merges a single typed key from branch states.
// The last branch to set the key wins.
func MergeTypedKey[T any](key Key[T]) Aggregator {
	return func(state *State, results map[string]*StepResult, errors map[string]error) error {
		for _, result := range results {
			if val, ok := GetFromBranch(result, key); ok {
				Set(state, key, val)
			}
		}
		return nil
	}
}

// CollectInto returns an aggregator that collects values from all branches into a slice.
// Each branch should set the inputKey; all values are collected into outputKey as []T.
// Order of collected values is non-deterministic due to concurrent execution.
func CollectInto[T any](inputKey Key[T], outputKey Key[[]T]) Aggregator {
	return func(state *State, results map[string]*StepResult, errors map[string]error) error {
		var collected []T
		for _, result := range results {
			if val, ok := GetFromBranch(result, inputKey); ok {
				collected = append(collected, val)
			}
		}
		Set(state, outputKey, collected)
		return nil
	}
}

// CollectMap returns an aggregator that collects values from all branches into a map.
// The map keys are the step names, values are the typed values from each branch.
func CollectMap[T any](inputKey Key[T], outputKey Key[map[string]T]) Aggregator {
	return func(state *State, results map[string]*StepResult, errors map[string]error) error {
		collected := make(map[string]T, len(results))
		for name, result := range results {
			if val, ok := GetFromBranch(result, inputKey); ok {
				collected[name] = val
			}
		}
		Set(state, outputKey, collected)
		return nil
	}
}

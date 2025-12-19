package workflow

// This file previously contained aggregator helpers for the old dynamic State system.
// With struct-based state, aggregators are custom functions that know the state schema:
//
// Example aggregator for struct-based state:
//
//	func collectResults(state *MyState, results map[string]*BranchResult[MyState], errors map[string]error) error {
//	    for name, br := range results {
//	        state.Results[name] = br.State.Result
//	    }
//	    return nil
//	}
//
// The Aggregator[S] type is defined in parallel.go.

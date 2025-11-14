package main

type sample struct {
	k string
	v *Item
}

// sampleKeys returns a slice of samples, each containing a key-value pair from the DB.
// The max number of samples is defined in the Config
func sampleKeys(state *AppState, expiring bool) []sample {
	maxSamples := state.conf.memSamples
	samples := make([]sample, 0, maxSamples)

	// Decide whether to grab from the normal or expiring store
	var store map[string]*Item
	if expiring {
		store = DB.expiringStore
	} else {
		store = DB.store
	}

	// Get a number of samples from the DB at most maxSamples
	for k, v := range store {
		samples = append(samples, sample{
			k: k,
			v: v,
		})
		if len(samples) >= maxSamples {
			break
		}
	}

	return samples
}

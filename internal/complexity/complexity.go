// Package complexity measures the complexity distributions of the measured source sets.
package complexity

import (
	"sort"

	"github.com/JacobJNilsson/scree/internal/contract"
	"github.com/JacobJNilsson/scree/internal/inventory"
)

const unitCount = "count"

// Measure returns the complexity metrics of both measured sets.
func Measure(inv *inventory.Inventory) map[string]contract.Metric {
	metrics := map[string]contract.Metric{}
	for _, set := range []contract.SourceSet{contract.Production, contract.Test} {
		for name, m := range measureSet(inv, set) {
			metrics[name+"."+string(set)] = m
		}
	}
	return metrics
}

// measureSet returns the metrics of one set, keyed by id without the set suffix.
func measureSet(inv *inventory.Inventory, set contract.SourceSet) map[string]contract.Metric {
	distribution := []string{"complexity.cc.p50", "complexity.cc.p90", "complexity.cc.max"}
	if inv.Incomplete(set) {
		detail := contract.Detail{Errors: errorPaths(inv, set)}
		metrics := map[string]contract.Metric{}
		for _, name := range append(distribution, "complexity.functions") {
			metrics[name] = contract.Metric{State: contract.Incomplete, Unit: unitCount, Detail: detail}
		}
		return metrics
	}
	var ccs []int
	for _, f := range inv.Functions {
		if f.Set == set {
			ccs = append(ccs, f.CC)
		}
	}
	sort.Ints(ccs)
	metrics := map[string]contract.Metric{"complexity.functions": count(float64(len(ccs)))}
	if len(ccs) == 0 {
		for _, name := range distribution {
			metrics[name] = contract.Metric{State: contract.NotApplicable, Unit: unitCount}
		}
		return metrics
	}
	metrics["complexity.cc.p50"] = count(float64(percentile(ccs, 50)))
	metrics["complexity.cc.p90"] = count(float64(percentile(ccs, 90)))
	metrics["complexity.cc.max"] = count(float64(ccs[len(ccs)-1]))
	return metrics
}

func count(value float64) contract.Metric {
	return contract.Metric{State: contract.Complete, Value: value, Unit: unitCount}
}

// errorPaths lists the paths of the errors that make a set incomplete, in the path order of the inventory.
func errorPaths(inv *inventory.Inventory, set contract.SourceSet) []string {
	var paths []string
	for _, e := range inv.Errors {
		if e.Set == set || (e.Kind == inventory.KindRead && e.Set == "") {
			paths = append(paths, e.Path)
		}
	}
	return paths
}

// percentile returns the nearest-rank percentile p of a non-empty ascending list, at the 1-based rank ceil(p/100 × n).
func percentile(sorted []int, p int) int {
	rank := (p*len(sorted) + 99) / 100
	return sorted[rank-1]
}

package contract

import (
	"slices"
	"testing"
)

func TestMetricIDsSortedPerSet(t *testing.T) {
	ids := MetricIDs()
	if len(ids) != 2*len(metricNames) || !slices.IsSorted(ids) {
		t.Errorf("MetricIDs() = %v, want one sorted id per name and set", ids)
	}
}

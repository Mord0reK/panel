package aggregation

import (
	"time"
)

type AggregationLevel struct {
	SourceTable         string
	TargetTable         string
	SourceThreshold     time.Duration
	AggregationInterval time.Duration
}

var ContainerAggregationLevels = []AggregationLevel{
	{
		SourceTable:         "metrics_5s",
		TargetTable:         "metrics_15s",
		SourceThreshold:     5 * time.Minute,
		AggregationInterval: 15 * time.Second,
	},
	{
		SourceTable:         "metrics_15s",
		TargetTable:         "metrics_30s",
		SourceThreshold:     15 * time.Minute,
		AggregationInterval: 30 * time.Second,
	},
	{
		SourceTable:         "metrics_30s",
		TargetTable:         "metrics_1m",
		SourceThreshold:     30 * time.Minute,
		AggregationInterval: 1 * time.Minute,
	},
	{
		SourceTable:         "metrics_1m",
		TargetTable:         "metrics_5m",
		SourceThreshold:     1 * time.Hour,
		AggregationInterval: 5 * time.Minute,
	},
	{
		SourceTable:         "metrics_5m",
		TargetTable:         "metrics_15m",
		SourceThreshold:     6 * time.Hour,
		AggregationInterval: 15 * time.Minute,
	},
	{
		SourceTable:         "metrics_15m",
		TargetTable:         "metrics_30m",
		SourceThreshold:     12 * time.Hour,
		AggregationInterval: 30 * time.Minute,
	},
	{
		SourceTable:         "metrics_30m",
		TargetTable:         "metrics_1h",
		SourceThreshold:     24 * time.Hour,
		AggregationInterval: 1 * time.Hour,
	},
	{
		SourceTable:         "metrics_1h",
		TargetTable:         "metrics_6h",
		SourceThreshold:     7 * 24 * time.Hour,
		AggregationInterval: 6 * time.Hour,
	},
	{
		SourceTable:         "metrics_6h",
		TargetTable:         "metrics_12h",
		SourceThreshold:     15 * 24 * time.Hour,
		AggregationInterval: 12 * time.Hour,
	},
}

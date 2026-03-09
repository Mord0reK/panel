package aggregation

import (
	"time"
)

type AggregationLevel struct {
	SourceTable         string
	TargetTable         string
	SourceThreshold     time.Duration
	AggregationInterval time.Duration
	RetentionThreshold  time.Duration
}

var ContainerAggregationLevels = []AggregationLevel{
	{
		SourceTable:         "metrics_5s",
		TargetTable:         "metrics_15s",
		SourceThreshold:     15 * time.Second,
		AggregationInterval: 15 * time.Second,
		RetentionThreshold:  5 * time.Minute,
	},
	{
		SourceTable:         "metrics_15s",
		TargetTable:         "metrics_30s",
		SourceThreshold:     30 * time.Second,
		AggregationInterval: 30 * time.Second,
		RetentionThreshold:  15 * time.Minute,
	},
	{
		SourceTable:         "metrics_30s",
		TargetTable:         "metrics_1m",
		SourceThreshold:     60 * time.Second,
		AggregationInterval: 1 * time.Minute,
		RetentionThreshold:  30 * time.Minute,
	},
	{
		SourceTable:         "metrics_1m",
		TargetTable:         "metrics_5m",
		SourceThreshold:     300 * time.Second,
		AggregationInterval: 5 * time.Minute,
		RetentionThreshold:  1 * time.Hour,
	},
	{
		SourceTable:         "metrics_5m",
		TargetTable:         "metrics_15m",
		SourceThreshold:     900 * time.Second,
		AggregationInterval: 15 * time.Minute,
		RetentionThreshold:  6 * time.Hour,
	},
	{
		SourceTable:         "metrics_15m",
		TargetTable:         "metrics_30m",
		SourceThreshold:     1800 * time.Second,
		AggregationInterval: 30 * time.Minute,
		RetentionThreshold:  12 * time.Hour,
	},
	{
		SourceTable:         "metrics_30m",
		TargetTable:         "metrics_1h",
		SourceThreshold:     3600 * time.Second,
		AggregationInterval: 1 * time.Hour,
		RetentionThreshold:  24 * time.Hour,
	},
	{
		SourceTable:         "metrics_1h",
		TargetTable:         "metrics_6h",
		SourceThreshold:     21600 * time.Second,
		AggregationInterval: 6 * time.Hour,
		RetentionThreshold:  7 * 24 * time.Hour,
	},
	{
		SourceTable:         "metrics_6h",
		TargetTable:         "metrics_12h",
		SourceThreshold:     43200 * time.Second,
		AggregationInterval: 12 * time.Hour,
		RetentionThreshold:  15 * 24 * time.Hour,
	},
}

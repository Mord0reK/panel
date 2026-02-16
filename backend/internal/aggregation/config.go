package aggregation

import (
	"time"
)

type AggregationLevel struct {
	SourceTable         string
	TargetTable         string
	SourceThreshold     time.Duration // Time after which data in SourceTable is aggregated
	AggregationInterval time.Duration // Group by this interval
	PointsToKeep        int           // How many points to keep in TargetTable (actually this logic is handled by next level deletion, but config says "keep 60")
	// "keep 60" in plan refers to keeping 60 points of current resolution?
	// Plan: "metrics_1s -> metrics_5s (> 1min, group by 5s, keep 60)"
	// This "keep 60" likely means we keep 60 points of metrics_5s? i.e. 5 minutes?
	// But metrics_5s feeds metrics_15s (> 5min).
	// So metrics_5s retains data for > 5min.
	// Actually, the plan says "metrics_1s -> metrics_5s (> 1min...)"
	// This implies we aggregate data OLDER than 1 min from 1s to 5s.
	// And we keep data in 1s for 1 min?
	// Let's stick to "Retention" logic.
	// "SourceThreshold" means "process data older than X".
	RetentionPeriod time.Duration // Data in SourceTable older than this is DELETED after aggregation?
	// Or maybe the plan "keep 60" means the retention of the SOURCE table?
	// "metrics_1s -> metrics_5s (> 1min, group by 5s, keep 60)"
	// This might mean: Retain 60 * 1s in metrics_1s.
	// But 60 * 1s = 1min.
	// So data older than 1min is aggregated to 5s and deleted from 1s.
	// Yes.
}

var ContainerAggregationLevels = []AggregationLevel{
	{
		SourceTable:         "metrics_1s",
		TargetTable:         "metrics_5s",
		SourceThreshold:     1 * time.Minute,
		AggregationInterval: 5 * time.Second,
	},
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

var HostAggregationLevels = []AggregationLevel{
	{
		SourceTable:         "host_metrics_1s",
		TargetTable:         "host_metrics_5s",
		SourceThreshold:     1 * time.Minute,
		AggregationInterval: 5 * time.Second,
	},
	{
		SourceTable:         "host_metrics_5s",
		TargetTable:         "host_metrics_15s",
		SourceThreshold:     5 * time.Minute,
		AggregationInterval: 15 * time.Second,
	},
	{
		SourceTable:         "host_metrics_15s",
		TargetTable:         "host_metrics_30s",
		SourceThreshold:     15 * time.Minute,
		AggregationInterval: 30 * time.Second,
	},
	{
		SourceTable:         "host_metrics_30s",
		TargetTable:         "host_metrics_1m",
		SourceThreshold:     30 * time.Minute,
		AggregationInterval: 1 * time.Minute,
	},
	{
		SourceTable:         "host_metrics_1m",
		TargetTable:         "host_metrics_5m",
		SourceThreshold:     1 * time.Hour,
		AggregationInterval: 5 * time.Minute,
	},
	{
		SourceTable:         "host_metrics_5m",
		TargetTable:         "host_metrics_15m",
		SourceThreshold:     6 * time.Hour,
		AggregationInterval: 15 * time.Minute,
	},
	{
		SourceTable:         "host_metrics_15m",
		TargetTable:         "host_metrics_30m",
		SourceThreshold:     12 * time.Hour,
		AggregationInterval: 30 * time.Minute,
	},
	{
		SourceTable:         "host_metrics_30m",
		TargetTable:         "host_metrics_1h",
		SourceThreshold:     24 * time.Hour,
		AggregationInterval: 1 * time.Hour,
	},
	{
		SourceTable:         "host_metrics_1h",
		TargetTable:         "host_metrics_6h",
		SourceThreshold:     7 * 24 * time.Hour,
		AggregationInterval: 6 * time.Hour,
	},
	{
		SourceTable:         "host_metrics_6h",
		TargetTable:         "host_metrics_12h",
		SourceThreshold:     15 * 24 * time.Hour,
		AggregationInterval: 12 * time.Hour,
	},
}

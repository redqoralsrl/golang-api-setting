package metrics

import "database/sql"

var (
	DatabaseConnections = NewGauge(
		"api_database_connections",
		"Current number of database connections.",
		"state",
	)

	DatabaseWaitCount = NewSimpleGauge(
		"api_database_wait_count",
		"Total number of database connection waits.",
	)

	DatabaseWaitDurationSeconds = NewSimpleGauge(
		"api_database_wait_duration_seconds",
		"Total time blocked waiting for a database connection in seconds.",
	)
)

func RecordDatabaseStats(stats sql.DBStats) {
	DatabaseConnections.WithLabelValues("idle").Set(float64(stats.Idle))
	DatabaseConnections.WithLabelValues("in_use").Set(float64(stats.InUse))
	DatabaseConnections.WithLabelValues("open").Set(float64(stats.OpenConnections))
	DatabaseWaitCount.Set(float64(stats.WaitCount))
	DatabaseWaitDurationSeconds.Set(stats.WaitDuration.Seconds())
}

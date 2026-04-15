package repository

import (
	"time"

	"github.com/trace-point/trace-point-renew/internal/domain"
)

// MetricsCacheRepo provides cache operations for deployment metrics.
type MetricsCacheRepo struct {
	db *DB
}

// NewMetricsCacheRepo creates a new MetricsCacheRepo.
func NewMetricsCacheRepo(db *DB) *MetricsCacheRepo {
	return &MetricsCacheRepo{db: db}
}

// Store caches deployment metrics data points.
func (r *MetricsCacheRepo) Store(metrics []domain.ContainerMetrics) error {
	tx, err := r.db.Conn().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO metrics_cache (deployment_name, namespace, cpu_percent, ram_percent, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, m := range metrics {
		_, err := stmt.Exec(m.DeploymentName, m.Namespace, m.CPUPercent, m.RAMPercent, m.Timestamp, now)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetHistory retrieves cached metrics for a deployment within a time range.
func (r *MetricsCacheRepo) GetHistory(deploymentName, namespace string, start, end time.Time) ([]domain.ContainerMetrics, error) {
	rows, err := r.db.Conn().Query(`
		SELECT deployment_name, namespace, cpu_percent, ram_percent, timestamp
		FROM metrics_cache
		WHERE deployment_name = ? AND namespace = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp ASC
	`, deploymentName, namespace, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []domain.ContainerMetrics
	for rows.Next() {
		var m domain.ContainerMetrics
		if err := rows.Scan(&m.DeploymentName, &m.Namespace, &m.CPUPercent, &m.RAMPercent, &m.Timestamp); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

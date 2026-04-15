package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trace-point/trace-point-renew/internal/domain"
)

// SpikeRepo provides CRUD operations for spike events.
type SpikeRepo struct {
	db *DB
}

// NewSpikeRepo creates a new SpikeRepo.
func NewSpikeRepo(db *DB) *SpikeRepo {
	return &SpikeRepo{db: db}
}

// Create inserts a new spike event.
func (r *SpikeRepo) Create(event *domain.SpikeEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	_, err := r.db.Conn().Exec(`
		INSERT INTO spike_events (
			id, timestamp, deployment_name, namespace,
			cpu_usage_percent, cpu_limit_percent, ram_usage_percent, ram_limit_percent,
			threshold_percent, moving_average_percent,
			route_name, trace_id, culprit_function, culprit_file_path,
			alert_sent, cooldown_end, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.ID, event.Timestamp, event.DeploymentName, event.Namespace,
		event.CPUUsagePercent, event.CPULimitPercent, event.RAMUsagePercent, event.RAMLimitPercent,
		event.ThresholdPercent, event.MovingAveragePercent,
		event.RouteName, event.TraceID, event.CulpritFunction, event.CulpritFilePath,
		event.AlertSent, event.CooldownEnd, event.CreatedAt,
	)
	return err
}

// GetByID retrieves a spike event by its UUID.
func (r *SpikeRepo) GetByID(id string) (*domain.SpikeEvent, error) {
	event := &domain.SpikeEvent{}
	err := r.db.Conn().QueryRow(`
		SELECT id, timestamp, deployment_name, namespace,
			cpu_usage_percent, cpu_limit_percent, ram_usage_percent, ram_limit_percent,
			threshold_percent, moving_average_percent,
			route_name, trace_id, culprit_function, culprit_file_path,
			alert_sent, cooldown_end, created_at
		FROM spike_events WHERE id = ?
	`, id).Scan(
		&event.ID, &event.Timestamp, &event.DeploymentName, &event.Namespace,
		&event.CPUUsagePercent, &event.CPULimitPercent, &event.RAMUsagePercent, &event.RAMLimitPercent,
		&event.ThresholdPercent, &event.MovingAveragePercent,
		&event.RouteName, &event.TraceID, &event.CulpritFunction, &event.CulpritFilePath,
		&event.AlertSent, &event.CooldownEnd, &event.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return event, err
}

// List retrieves spike events with optional filtering and pagination.
func (r *SpikeRepo) List(filter domain.SpikeListFilter) ([]domain.SpikeEvent, int, error) {
	var conditions []string
	var args []interface{}

	if filter.Namespace != "" {
		conditions = append(conditions, "namespace = ?")
		args = append(args, filter.Namespace)
	}
	if filter.DeploymentName != "" {
		conditions = append(conditions, "deployment_name = ?")
		args = append(args, filter.DeploymentName)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM spike_events %s", whereClause)
	if err := r.db.Conn().QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build sort clause
	sortColumn := "timestamp"
	switch filter.Sort {
	case "cpu":
		sortColumn = "cpu_usage_percent"
	case "ram":
		sortColumn = "ram_usage_percent"
	case "deployment":
		sortColumn = "deployment_name"
	case "time":
		sortColumn = "timestamp"
	}

	order := "DESC"
	if filter.Order == "asc" {
		order = "ASC"
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, timestamp, deployment_name, namespace,
			cpu_usage_percent, cpu_limit_percent, ram_usage_percent, ram_limit_percent,
			threshold_percent, moving_average_percent,
			route_name, trace_id, culprit_function, culprit_file_path,
			alert_sent, cooldown_end, created_at
		FROM spike_events %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, sortColumn, order)

	queryArgs := append(args, limit, offset)
	rows, err := r.db.Conn().Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []domain.SpikeEvent
	for rows.Next() {
		var e domain.SpikeEvent
		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.DeploymentName, &e.Namespace,
			&e.CPUUsagePercent, &e.CPULimitPercent, &e.RAMUsagePercent, &e.RAMLimitPercent,
			&e.ThresholdPercent, &e.MovingAveragePercent,
			&e.RouteName, &e.TraceID, &e.CulpritFunction, &e.CulpritFilePath,
			&e.AlertSent, &e.CooldownEnd, &e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}

	return events, total, rows.Err()
}

// Update updates a spike event (e.g., after correlation completes).
func (r *SpikeRepo) Update(event *domain.SpikeEvent) error {
	_, err := r.db.Conn().Exec(`
		UPDATE spike_events SET
			route_name = ?, trace_id = ?, culprit_function = ?, culprit_file_path = ?,
			alert_sent = ?, cooldown_end = ?
		WHERE id = ?
	`,
		event.RouteName, event.TraceID, event.CulpritFunction, event.CulpritFilePath,
		event.AlertSent, event.CooldownEnd, event.ID,
	)
	return err
}

// GetRecentByDeployment returns the most recent spike for a deployment (for cooldown check).
func (r *SpikeRepo) GetRecentByDeployment(deploymentName, namespace string) (*domain.SpikeEvent, error) {
	event := &domain.SpikeEvent{}
	err := r.db.Conn().QueryRow(`
		SELECT id, timestamp, deployment_name, namespace,
			cpu_usage_percent, cpu_limit_percent, ram_usage_percent, ram_limit_percent,
			threshold_percent, moving_average_percent,
			route_name, trace_id, culprit_function, culprit_file_path,
			alert_sent, cooldown_end, created_at
		FROM spike_events
		WHERE deployment_name = ? AND namespace = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`, deploymentName, namespace).Scan(
		&event.ID, &event.Timestamp, &event.DeploymentName, &event.Namespace,
		&event.CPUUsagePercent, &event.CPULimitPercent, &event.RAMUsagePercent, &event.RAMLimitPercent,
		&event.ThresholdPercent, &event.MovingAveragePercent,
		&event.RouteName, &event.TraceID, &event.CulpritFunction, &event.CulpritFilePath,
		&event.AlertSent, &event.CooldownEnd, &event.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return event, err
}

// GetSpikesForGravity returns aggregated spike data for gravity score calculation.
func (r *SpikeRepo) GetSpikesForGravity(days int) ([]domain.SpikeEvent, error) {
	cutoff := time.Now().AddDate(0, 0, -days)

	rows, err := r.db.Conn().Query(`
		SELECT id, timestamp, deployment_name, namespace,
			cpu_usage_percent, cpu_limit_percent, ram_usage_percent, ram_limit_percent,
			threshold_percent, moving_average_percent,
			route_name, trace_id, culprit_function, culprit_file_path,
			alert_sent, cooldown_end, created_at
		FROM spike_events
		WHERE created_at >= ?
		ORDER BY timestamp DESC
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.SpikeEvent
	for rows.Next() {
		var e domain.SpikeEvent
		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.DeploymentName, &e.Namespace,
			&e.CPUUsagePercent, &e.CPULimitPercent, &e.RAMUsagePercent, &e.RAMLimitPercent,
			&e.ThresholdPercent, &e.MovingAveragePercent,
			&e.RouteName, &e.TraceID, &e.CulpritFunction, &e.CulpritFilePath,
			&e.AlertSent, &e.CooldownEnd, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// GetAllForExport returns all spike events within the retention period for JSON export.
func (r *SpikeRepo) GetAllForExport(days int) ([]domain.SpikeEvent, error) {
	return r.GetSpikesForGravity(days)
}

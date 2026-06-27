package model

import (
	"context"
	"database/sql"
	"time"
)

type AdminDashboardTask struct {
	Type      string
	Title     string
	CityName  string
	CreatedAt string
}

type AdminDashboardOverview struct {
	PendingResourceCount     int64
	PendingVerificationCount int64
	PendingDemandCount       int64
	TodayContactCount        int64
	Tasks                    []AdminDashboardTask
}

type AdminDashboardModel struct {
	db *sql.DB
}

func NewAdminDashboardModel(db *sql.DB) *AdminDashboardModel {
	return &AdminDashboardModel{db: db}
}

func (m *AdminDashboardModel) GetAdminDashboardOverview(ctx context.Context, cityCode string) (AdminDashboardOverview, error) {
	var overview AdminDashboardOverview
	err := m.db.QueryRowContext(ctx, `
SELECT
  (SELECT COUNT(*) FROM resources r JOIN city_stations cs ON cs.id = r.city_station_id WHERE r.status = 'pending' AND r.deleted_at IS NULL AND ($1 = '' OR cs.code = $1)),
  (SELECT COUNT(*) FROM verifications v JOIN merchants m ON m.id = v.merchant_id JOIN city_stations cs ON cs.id = m.city_station_id WHERE v.status = 'pending' AND ($1 = '' OR cs.code = $1)),
  (SELECT COUNT(*) FROM purchase_demands pd LEFT JOIN city_stations cs ON cs.id = pd.city_station_id WHERE pd.status IN ('pending', 'matching') AND ($1 = '' OR cs.code = $1)),
  (SELECT COUNT(*) FROM resource_contact_events rce JOIN resources r ON r.id = rce.resource_id JOIN city_stations cs ON cs.id = r.city_station_id WHERE rce.created_at >= CURRENT_DATE AND ($1 = '' OR cs.code = $1))
`, cityCode).Scan(&overview.PendingResourceCount, &overview.PendingVerificationCount, &overview.PendingDemandCount, &overview.TodayContactCount)
	if err != nil {
		return AdminDashboardOverview{}, err
	}

	rows, err := m.db.QueryContext(ctx, `
SELECT type, title, city_name, created_at
FROM (
  SELECT '资源审核' AS type, r.title AS title, cs.name AS city_name, r.created_at AS created_at
  FROM resources r
  JOIN city_stations cs ON cs.id = r.city_station_id
  WHERE r.status = 'pending' AND r.deleted_at IS NULL AND ($1 = '' OR cs.code = $1)
  UNION ALL
  SELECT '认证审核' AS type, COALESCE(m.name, v.verification_type) AS title, cs.name AS city_name, v.submitted_at AS created_at
  FROM verifications v
  JOIN merchants m ON m.id = v.merchant_id
  JOIN city_stations cs ON cs.id = m.city_station_id
  WHERE v.status = 'pending' AND ($1 = '' OR cs.code = $1)
  UNION ALL
  SELECT '采购需求' AS type, pd.title AS title, COALESCE(cs.name, '') AS city_name, pd.created_at AS created_at
  FROM purchase_demands pd
  LEFT JOIN city_stations cs ON cs.id = pd.city_station_id
  WHERE pd.status IN ('pending', 'matching') AND ($1 = '' OR cs.code = $1)
) pending_tasks
ORDER BY created_at DESC
LIMIT 10
`, cityCode)
	if err != nil {
		return AdminDashboardOverview{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var task AdminDashboardTask
		var createdAt time.Time
		if err := rows.Scan(&task.Type, &task.Title, &task.CityName, &createdAt); err != nil {
			return AdminDashboardOverview{}, err
		}
		task.CreatedAt = createdAt.Format(time.RFC3339)
		overview.Tasks = append(overview.Tasks, task)
	}
	if err := rows.Err(); err != nil {
		return AdminDashboardOverview{}, err
	}
	return overview, nil
}

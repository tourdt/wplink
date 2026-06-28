package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ ResourceMetricsDailyModel = (*customResourceMetricsDailyModel)(nil)

type (
	// ResourceMetricsDailyModel is an interface to be customized, add more methods here,
	// and implement the added methods in customResourceMetricsDailyModel.
	ResourceMetricsDailyModel interface {
		resourceMetricsDailyModel
		withSession(session sqlx.Session) ResourceMetricsDailyModel
	}

	customResourceMetricsDailyModel struct {
		*defaultResourceMetricsDailyModel
	}
)

// NewResourceMetricsDailyModel returns a model for the database table.
func NewResourceMetricsDailyModel(conn sqlx.SqlConn) ResourceMetricsDailyModel {
	return &customResourceMetricsDailyModel{
		defaultResourceMetricsDailyModel: newResourceMetricsDailyModel(conn),
	}
}

func (m *customResourceMetricsDailyModel) withSession(session sqlx.Session) ResourceMetricsDailyModel {
	return NewResourceMetricsDailyModel(sqlx.NewSqlConnFromSession(session))
}

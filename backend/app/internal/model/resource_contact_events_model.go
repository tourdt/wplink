package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ ResourceContactEventsModel = (*customResourceContactEventsModel)(nil)

type (
	// ResourceContactEventsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customResourceContactEventsModel.
	ResourceContactEventsModel interface {
		resourceContactEventsModel
		withSession(session sqlx.Session) ResourceContactEventsModel
	}

	customResourceContactEventsModel struct {
		*defaultResourceContactEventsModel
	}
)

// NewResourceContactEventsModel returns a model for the database table.
func NewResourceContactEventsModel(conn sqlx.SqlConn) ResourceContactEventsModel {
	return &customResourceContactEventsModel{
		defaultResourceContactEventsModel: newResourceContactEventsModel(conn),
	}
}

func (m *customResourceContactEventsModel) withSession(session sqlx.Session) ResourceContactEventsModel {
	return NewResourceContactEventsModel(sqlx.NewSqlConnFromSession(session))
}

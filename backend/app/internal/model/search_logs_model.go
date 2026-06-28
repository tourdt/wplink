package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ SearchLogsModel = (*customSearchLogsModel)(nil)

type (
	// SearchLogsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSearchLogsModel.
	SearchLogsModel interface {
		searchLogsModel
		withSession(session sqlx.Session) SearchLogsModel
	}

	customSearchLogsModel struct {
		*defaultSearchLogsModel
	}
)

// NewSearchLogsModel returns a model for the database table.
func NewSearchLogsModel(conn sqlx.SqlConn) SearchLogsModel {
	return &customSearchLogsModel{
		defaultSearchLogsModel: newSearchLogsModel(conn),
	}
}

func (m *customSearchLogsModel) withSession(session sqlx.Session) SearchLogsModel {
	return NewSearchLogsModel(sqlx.NewSqlConnFromSession(session))
}

package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ VerificationsModel = (*customVerificationsModel)(nil)

type (
	// VerificationsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customVerificationsModel.
	VerificationsModel interface {
		verificationsModel
		withSession(session sqlx.Session) VerificationsModel
	}

	customVerificationsModel struct {
		*defaultVerificationsModel
	}
)

// NewVerificationsModel returns a model for the database table.
func NewVerificationsModel(conn sqlx.SqlConn) VerificationsModel {
	return &customVerificationsModel{
		defaultVerificationsModel: newVerificationsModel(conn),
	}
}

func (m *customVerificationsModel) withSession(session sqlx.Session) VerificationsModel {
	return NewVerificationsModel(sqlx.NewSqlConnFromSession(session))
}

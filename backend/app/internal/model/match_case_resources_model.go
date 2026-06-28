package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ MatchCaseResourcesModel = (*customMatchCaseResourcesModel)(nil)

type (
	// MatchCaseResourcesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMatchCaseResourcesModel.
	MatchCaseResourcesModel interface {
		matchCaseResourcesModel
		withSession(session sqlx.Session) MatchCaseResourcesModel
	}

	customMatchCaseResourcesModel struct {
		*defaultMatchCaseResourcesModel
	}
)

// NewMatchCaseResourcesModel returns a model for the database table.
func NewMatchCaseResourcesModel(conn sqlx.SqlConn) MatchCaseResourcesModel {
	return &customMatchCaseResourcesModel{
		defaultMatchCaseResourcesModel: newMatchCaseResourcesModel(conn),
	}
}

func (m *customMatchCaseResourcesModel) withSession(session sqlx.Session) MatchCaseResourcesModel {
	return NewMatchCaseResourcesModel(sqlx.NewSqlConnFromSession(session))
}

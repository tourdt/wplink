package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ MatchCasesModel = (*customMatchCasesModel)(nil)

type (
	// MatchCasesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMatchCasesModel.
	MatchCasesModel interface {
		matchCasesModel
		withSession(session sqlx.Session) MatchCasesModel
	}

	customMatchCasesModel struct {
		*defaultMatchCasesModel
	}
)

// NewMatchCasesModel returns a model for the database table.
func NewMatchCasesModel(conn sqlx.SqlConn) MatchCasesModel {
	return &customMatchCasesModel{
		defaultMatchCasesModel: newMatchCasesModel(conn),
	}
}

func (m *customMatchCasesModel) withSession(session sqlx.Session) MatchCasesModel {
	return NewMatchCasesModel(sqlx.NewSqlConnFromSession(session))
}

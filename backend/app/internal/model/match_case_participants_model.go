package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ MatchCaseParticipantsModel = (*customMatchCaseParticipantsModel)(nil)

type (
	// MatchCaseParticipantsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMatchCaseParticipantsModel.
	MatchCaseParticipantsModel interface {
		matchCaseParticipantsModel
		withSession(session sqlx.Session) MatchCaseParticipantsModel
	}

	customMatchCaseParticipantsModel struct {
		*defaultMatchCaseParticipantsModel
	}
)

// NewMatchCaseParticipantsModel returns a model for the database table.
func NewMatchCaseParticipantsModel(conn sqlx.SqlConn) MatchCaseParticipantsModel {
	return &customMatchCaseParticipantsModel{
		defaultMatchCaseParticipantsModel: newMatchCaseParticipantsModel(conn),
	}
}

func (m *customMatchCaseParticipantsModel) withSession(session sqlx.Session) MatchCaseParticipantsModel {
	return NewMatchCaseParticipantsModel(sqlx.NewSqlConnFromSession(session))
}

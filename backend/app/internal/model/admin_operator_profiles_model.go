package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ AdminOperatorProfilesModel = (*customAdminOperatorProfilesModel)(nil)

type (
	// AdminOperatorProfilesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAdminOperatorProfilesModel.
	AdminOperatorProfilesModel interface {
		adminOperatorProfilesModel
		withSession(session sqlx.Session) AdminOperatorProfilesModel
	}

	customAdminOperatorProfilesModel struct {
		*defaultAdminOperatorProfilesModel
	}
)

// NewAdminOperatorProfilesModel returns a model for the database table.
func NewAdminOperatorProfilesModel(conn sqlx.SqlConn) AdminOperatorProfilesModel {
	return &customAdminOperatorProfilesModel{
		defaultAdminOperatorProfilesModel: newAdminOperatorProfilesModel(conn),
	}
}

func (m *customAdminOperatorProfilesModel) withSession(session sqlx.Session) AdminOperatorProfilesModel {
	return NewAdminOperatorProfilesModel(sqlx.NewSqlConnFromSession(session))
}

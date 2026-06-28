package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ AdminLoginCredentialsModel = (*customAdminLoginCredentialsModel)(nil)

type (
	// AdminLoginCredentialsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAdminLoginCredentialsModel.
	AdminLoginCredentialsModel interface {
		adminLoginCredentialsModel
		withSession(session sqlx.Session) AdminLoginCredentialsModel
	}

	customAdminLoginCredentialsModel struct {
		*defaultAdminLoginCredentialsModel
	}
)

// NewAdminLoginCredentialsModel returns a model for the database table.
func NewAdminLoginCredentialsModel(conn sqlx.SqlConn) AdminLoginCredentialsModel {
	return &customAdminLoginCredentialsModel{
		defaultAdminLoginCredentialsModel: newAdminLoginCredentialsModel(conn),
	}
}

func (m *customAdminLoginCredentialsModel) withSession(session sqlx.Session) AdminLoginCredentialsModel {
	return NewAdminLoginCredentialsModel(sqlx.NewSqlConnFromSession(session))
}

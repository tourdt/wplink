package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ UserRoleAssignmentsModel = (*customUserRoleAssignmentsModel)(nil)

type (
	// UserRoleAssignmentsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserRoleAssignmentsModel.
	UserRoleAssignmentsModel interface {
		userRoleAssignmentsModel
		withSession(session sqlx.Session) UserRoleAssignmentsModel
	}

	customUserRoleAssignmentsModel struct {
		*defaultUserRoleAssignmentsModel
	}
)

// NewUserRoleAssignmentsModel returns a model for the database table.
func NewUserRoleAssignmentsModel(conn sqlx.SqlConn) UserRoleAssignmentsModel {
	return &customUserRoleAssignmentsModel{
		defaultUserRoleAssignmentsModel: newUserRoleAssignmentsModel(conn),
	}
}

func (m *customUserRoleAssignmentsModel) withSession(session sqlx.Session) UserRoleAssignmentsModel {
	return NewUserRoleAssignmentsModel(sqlx.NewSqlConnFromSession(session))
}

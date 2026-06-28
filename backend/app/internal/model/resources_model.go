package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ ResourcesModel = (*customResourcesModel)(nil)

type (
	// ResourcesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customResourcesModel.
	ResourcesModel interface {
		resourcesModel
		withSession(session sqlx.Session) ResourcesModel
	}

	customResourcesModel struct {
		*defaultResourcesModel
	}
)

// NewResourcesModel returns a model for the database table.
func NewResourcesModel(conn sqlx.SqlConn) ResourcesModel {
	return &customResourcesModel{
		defaultResourcesModel: newResourcesModel(conn),
	}
}

func (m *customResourcesModel) withSession(session sqlx.Session) ResourcesModel {
	return NewResourcesModel(sqlx.NewSqlConnFromSession(session))
}

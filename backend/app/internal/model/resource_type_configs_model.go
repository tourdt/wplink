package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ ResourceTypeConfigsModel = (*customResourceTypeConfigsModel)(nil)

type (
	// ResourceTypeConfigsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customResourceTypeConfigsModel.
	ResourceTypeConfigsModel interface {
		resourceTypeConfigsModel
		withSession(session sqlx.Session) ResourceTypeConfigsModel
	}

	customResourceTypeConfigsModel struct {
		*defaultResourceTypeConfigsModel
	}
)

// NewResourceTypeConfigsModel returns a model for the database table.
func NewResourceTypeConfigsModel(conn sqlx.SqlConn) ResourceTypeConfigsModel {
	return &customResourceTypeConfigsModel{
		defaultResourceTypeConfigsModel: newResourceTypeConfigsModel(conn),
	}
}

func (m *customResourceTypeConfigsModel) withSession(session sqlx.Session) ResourceTypeConfigsModel {
	return NewResourceTypeConfigsModel(sqlx.NewSqlConnFromSession(session))
}

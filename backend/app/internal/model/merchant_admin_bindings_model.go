package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ MerchantAdminBindingsModel = (*customMerchantAdminBindingsModel)(nil)

type (
	// MerchantAdminBindingsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMerchantAdminBindingsModel.
	MerchantAdminBindingsModel interface {
		merchantAdminBindingsModel
		withSession(session sqlx.Session) MerchantAdminBindingsModel
	}

	customMerchantAdminBindingsModel struct {
		*defaultMerchantAdminBindingsModel
	}
)

// NewMerchantAdminBindingsModel returns a model for the database table.
func NewMerchantAdminBindingsModel(conn sqlx.SqlConn) MerchantAdminBindingsModel {
	return &customMerchantAdminBindingsModel{
		defaultMerchantAdminBindingsModel: newMerchantAdminBindingsModel(conn),
	}
}

func (m *customMerchantAdminBindingsModel) withSession(session sqlx.Session) MerchantAdminBindingsModel {
	return NewMerchantAdminBindingsModel(sqlx.NewSqlConnFromSession(session))
}

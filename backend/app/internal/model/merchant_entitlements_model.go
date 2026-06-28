package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ MerchantEntitlementsModel = (*customMerchantEntitlementsModel)(nil)

type (
	// MerchantEntitlementsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMerchantEntitlementsModel.
	MerchantEntitlementsModel interface {
		merchantEntitlementsModel
		withSession(session sqlx.Session) MerchantEntitlementsModel
	}

	customMerchantEntitlementsModel struct {
		*defaultMerchantEntitlementsModel
	}
)

// NewMerchantEntitlementsModel returns a model for the database table.
func NewMerchantEntitlementsModel(conn sqlx.SqlConn) MerchantEntitlementsModel {
	return &customMerchantEntitlementsModel{
		defaultMerchantEntitlementsModel: newMerchantEntitlementsModel(conn),
	}
}

func (m *customMerchantEntitlementsModel) withSession(session sqlx.Session) MerchantEntitlementsModel {
	return NewMerchantEntitlementsModel(sqlx.NewSqlConnFromSession(session))
}

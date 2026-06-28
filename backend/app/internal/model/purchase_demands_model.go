package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ PurchaseDemandsModel = (*customPurchaseDemandsModel)(nil)

type (
	// PurchaseDemandsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPurchaseDemandsModel.
	PurchaseDemandsModel interface {
		purchaseDemandsModel
		withSession(session sqlx.Session) PurchaseDemandsModel
	}

	customPurchaseDemandsModel struct {
		*defaultPurchaseDemandsModel
	}
)

// NewPurchaseDemandsModel returns a model for the database table.
func NewPurchaseDemandsModel(conn sqlx.SqlConn) PurchaseDemandsModel {
	return &customPurchaseDemandsModel{
		defaultPurchaseDemandsModel: newPurchaseDemandsModel(conn),
	}
}

func (m *customPurchaseDemandsModel) withSession(session sqlx.Session) PurchaseDemandsModel {
	return NewPurchaseDemandsModel(sqlx.NewSqlConnFromSession(session))
}

package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ MerchantLocationReportsModel = (*customMerchantLocationReportsModel)(nil)

type (
	// MerchantLocationReportsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMerchantLocationReportsModel.
	MerchantLocationReportsModel interface {
		merchantLocationReportsModel
		withSession(session sqlx.Session) MerchantLocationReportsModel
	}

	customMerchantLocationReportsModel struct {
		*defaultMerchantLocationReportsModel
	}
)

// NewMerchantLocationReportsModel returns a model for the database table.
func NewMerchantLocationReportsModel(conn sqlx.SqlConn) MerchantLocationReportsModel {
	return &customMerchantLocationReportsModel{
		defaultMerchantLocationReportsModel: newMerchantLocationReportsModel(conn),
	}
}

func (m *customMerchantLocationReportsModel) withSession(session sqlx.Session) MerchantLocationReportsModel {
	return NewMerchantLocationReportsModel(sqlx.NewSqlConnFromSession(session))
}

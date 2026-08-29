package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ MerchantDirectoryEventsModel = (*customMerchantDirectoryEventsModel)(nil)

type (
	// MerchantDirectoryEventsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMerchantDirectoryEventsModel.
	MerchantDirectoryEventsModel interface {
		merchantDirectoryEventsModel
		withSession(session sqlx.Session) MerchantDirectoryEventsModel
	}

	customMerchantDirectoryEventsModel struct {
		*defaultMerchantDirectoryEventsModel
	}
)

// NewMerchantDirectoryEventsModel returns a model for the database table.
func NewMerchantDirectoryEventsModel(conn sqlx.SqlConn) MerchantDirectoryEventsModel {
	return &customMerchantDirectoryEventsModel{
		defaultMerchantDirectoryEventsModel: newMerchantDirectoryEventsModel(conn),
	}
}

func (m *customMerchantDirectoryEventsModel) withSession(session sqlx.Session) MerchantDirectoryEventsModel {
	return NewMerchantDirectoryEventsModel(sqlx.NewSqlConnFromSession(session))
}

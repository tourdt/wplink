package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ TopVouchersModel = (*customTopVouchersModel)(nil)

type (
	// TopVouchersModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTopVouchersModel.
	TopVouchersModel interface {
		topVouchersModel
		withSession(session sqlx.Session) TopVouchersModel
	}

	customTopVouchersModel struct {
		*defaultTopVouchersModel
	}
)

// NewTopVouchersModel returns a model for the database table.
func NewTopVouchersModel(conn sqlx.SqlConn) TopVouchersModel {
	return &customTopVouchersModel{
		defaultTopVouchersModel: newTopVouchersModel(conn),
	}
}

func (m *customTopVouchersModel) withSession(session sqlx.Session) TopVouchersModel {
	return NewTopVouchersModel(sqlx.NewSqlConnFromSession(session))
}

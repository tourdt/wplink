package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ CreditRecordsModel = (*customCreditRecordsModel)(nil)

type (
	// CreditRecordsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCreditRecordsModel.
	CreditRecordsModel interface {
		creditRecordsModel
		withSession(session sqlx.Session) CreditRecordsModel
	}

	customCreditRecordsModel struct {
		*defaultCreditRecordsModel
	}
)

// NewCreditRecordsModel returns a model for the database table.
func NewCreditRecordsModel(conn sqlx.SqlConn) CreditRecordsModel {
	return &customCreditRecordsModel{
		defaultCreditRecordsModel: newCreditRecordsModel(conn),
	}
}

func (m *customCreditRecordsModel) withSession(session sqlx.Session) CreditRecordsModel {
	return NewCreditRecordsModel(sqlx.NewSqlConnFromSession(session))
}

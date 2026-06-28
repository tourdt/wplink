package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ ResourceReviewRecordsModel = (*customResourceReviewRecordsModel)(nil)

type (
	// ResourceReviewRecordsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customResourceReviewRecordsModel.
	ResourceReviewRecordsModel interface {
		resourceReviewRecordsModel
		withSession(session sqlx.Session) ResourceReviewRecordsModel
	}

	customResourceReviewRecordsModel struct {
		*defaultResourceReviewRecordsModel
	}
)

// NewResourceReviewRecordsModel returns a model for the database table.
func NewResourceReviewRecordsModel(conn sqlx.SqlConn) ResourceReviewRecordsModel {
	return &customResourceReviewRecordsModel{
		defaultResourceReviewRecordsModel: newResourceReviewRecordsModel(conn),
	}
}

func (m *customResourceReviewRecordsModel) withSession(session sqlx.Session) ResourceReviewRecordsModel {
	return NewResourceReviewRecordsModel(sqlx.NewSqlConnFromSession(session))
}

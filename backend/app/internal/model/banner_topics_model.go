package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ BannerTopicsModel = (*customBannerTopicsModel)(nil)

type (
	// BannerTopicsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customBannerTopicsModel.
	BannerTopicsModel interface {
		bannerTopicsModel
		withSession(session sqlx.Session) BannerTopicsModel
	}

	customBannerTopicsModel struct {
		*defaultBannerTopicsModel
	}
)

// NewBannerTopicsModel returns a model for the database table.
func NewBannerTopicsModel(conn sqlx.SqlConn) BannerTopicsModel {
	return &customBannerTopicsModel{
		defaultBannerTopicsModel: newBannerTopicsModel(conn),
	}
}

func (m *customBannerTopicsModel) withSession(session sqlx.Session) BannerTopicsModel {
	return NewBannerTopicsModel(sqlx.NewSqlConnFromSession(session))
}

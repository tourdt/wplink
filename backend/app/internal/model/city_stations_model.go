package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ CityStationsModel = (*customCityStationsModel)(nil)

type (
	// CityStationsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCityStationsModel.
	CityStationsModel interface {
		cityStationsModel
		withSession(session sqlx.Session) CityStationsModel
	}

	customCityStationsModel struct {
		*defaultCityStationsModel
	}
)

// NewCityStationsModel returns a model for the database table.
func NewCityStationsModel(conn sqlx.SqlConn) CityStationsModel {
	return &customCityStationsModel{
		defaultCityStationsModel: newCityStationsModel(conn),
	}
}

func (m *customCityStationsModel) withSession(session sqlx.Session) CityStationsModel {
	return NewCityStationsModel(sqlx.NewSqlConnFromSession(session))
}

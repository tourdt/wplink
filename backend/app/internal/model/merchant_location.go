package model

import "math"

const (
	MerchantLocationActionSet        = "set"
	MerchantLocationActionClear      = "clear"
	MerchantLocationActionKeep       = "keep"
	MerchantLocationStatusVisible    = "visible"
	MerchantLocationStatusHidden     = "hidden"
	MerchantLocationStatusRiskHidden = "risk_hidden"
)

type MerchantLocation struct {
	MarketName   string
	BuildingName string
	FloorNo      string
	BoothNo      string
	Latitude     *float64
	Longitude    *float64
	LocationName string
	Status       string
	UpdatedAt    string
}

type MerchantLocationPatch struct {
	Action       string
	MarketName   string
	BuildingName string
	FloorNo      string
	BoothNo      string
	Latitude     float64
	Longitude    float64
	LocationName string
	Status       string
}

// ValidMerchantCoordinates 统一写入、主页与位置治理的坐标边界，合法 0 值不能作为“未选位置”。
// 是否允许公开仍由主体、资料和位置状态分别决定，坐标有效本身不代表商家已入驻。
func ValidMerchantCoordinates(latitude, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsInf(latitude, 0) && latitude >= -90 && latitude <= 90 &&
		!math.IsNaN(longitude) && !math.IsInf(longitude, 0) && longitude >= -180 && longitude <= 180
}

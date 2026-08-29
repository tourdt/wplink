package model

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBuildMerchantDirectoryWhereOnlyReturnsVisibleCompletedMerchants(t *testing.T) {
	where, args := buildMerchantDirectoryWhere(MerchantDirectoryFilter{
		CityCode: "zhili", Keyword: "A108", Categories: []string{"童装"},
	})

	for _, token := range []string{
		"m.deleted_at IS NULL",
		"m.status = 'active'",
		"m.profile_status = 'completed'",
		"m.location_status = 'visible'",
		"m.latitude IS NOT NULL",
		"m.longitude IS NOT NULL",
		"m.name ILIKE",
		"m.market_name ILIKE",
		"m.building_name ILIKE",
		"m.floor_no ILIKE",
		"m.booth_no ILIKE",
		"m.main_categories ?|",
	} {
		if !strings.Contains(where, token) {
			t.Fatalf("where = %q, missing %q", where, token)
		}
	}
	if strings.Contains(where, "A108") {
		t.Fatalf("where = %q, keyword must be bound instead of interpolated", where)
	}
	if len(args) != 3 {
		t.Fatalf("args = %#v, want city, keyword and categories", args)
	}
}

func TestBuildMerchantDirectoryWhereTreatsKeywordAsData(t *testing.T) {
	keyword := `A108%' OR TRUE --`
	where, args := buildMerchantDirectoryWhere(MerchantDirectoryFilter{CityCode: "zhili", Keyword: keyword})

	if strings.Contains(where, keyword) {
		t.Fatalf("where = %q, must not interpolate untrusted keyword", where)
	}
	if len(args) != 2 || args[1] != keyword {
		t.Fatalf("args = %#v, want raw keyword in bound argument", args)
	}
}

func TestRankNearbyVisibleMerchantsFiltersSortsAndLimits(t *testing.T) {
	origin := MerchantDirectoryItem{MerchantID: "merchant-origin", Latitude: 30.8700, Longitude: 120.1200}
	candidates := make([]MerchantDirectoryItem, 0, 24)
	for index := 23; index >= 0; index-- {
		// 每 0.0001 纬度约 11 米，24 家候选均在一公里内；倒序输入用于证明结果按距离排序。
		candidates = append(candidates, MerchantDirectoryItem{
			MerchantID: "merchant-" + string(rune('a'+index)),
			Latitude:   30.8700 + float64(index+1)*0.0001,
			Longitude:  120.1200,
		})
	}
	candidates = append(candidates,
		origin,
		MerchantDirectoryItem{MerchantID: "merchant-far", Latitude: 30.8900, Longitude: 120.1200},
		MerchantDirectoryItem{MerchantID: "merchant-invalid", Latitude: math.NaN(), Longitude: 120.1200},
	)

	items := rankNearbyVisibleMerchants(origin, candidates, 1000, 20)

	if len(items) != 20 {
		t.Fatalf("len(items) = %d, want 20", len(items))
	}
	for index, item := range items {
		if item.MerchantID == origin.MerchantID || item.MerchantID == "merchant-far" || item.MerchantID == "merchant-invalid" {
			t.Fatalf("items[%d] = %#v, want only other valid merchants inside radius", index, item)
		}
		if item.DistanceMeters <= 0 || item.DistanceMeters > 1000 {
			t.Fatalf("items[%d].DistanceMeters = %f, want (0, 1000]", index, item.DistanceMeters)
		}
		if index > 0 && items[index-1].DistanceMeters > item.DistanceMeters {
			t.Fatalf("items are not ordered by distance: %#v", items)
		}
	}
}

func TestBuildNearbyVisibleMerchantsQueryUsesBoundingBoxAndPublicConditions(t *testing.T) {
	origin := MerchantDirectoryItem{MerchantID: "801", Latitude: 30.87, Longitude: 120.12}
	query, args := buildNearbyVisibleMerchantsQuery(origin, 1000)

	for _, token := range []string{
		"m.status = 'active'",
		"m.profile_status = 'completed'",
		"m.location_status = 'visible'",
		"cs.code = 'zhili'",
		"m.id <> $1::bigint",
		"m.latitude BETWEEN $2 AND $3",
		"m.longitude BETWEEN $4 AND $5",
	} {
		if !strings.Contains(query, token) {
			t.Fatalf("query = %q, missing %q", query, token)
		}
	}
	if len(args) != 5 || args[0] != "801" {
		t.Fatalf("args = %#v, want merchant ID and four bounding values", args)
	}
}

func TestGetVisibleMerchantLocationOnlyReadsZhiliDirectoryMerchant(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`(?s)FROM merchants m.*JOIN city_stations cs.*WHERE m\.id = \$1::bigint.*cs\.code = 'zhili'.*m\.status = 'active'.*m\.profile_status = 'completed'.*m\.location_status = 'visible'`).
		WithArgs("801").
		WillReturnRows(sqlmock.NewRows(merchantDirectoryModelTestColumns()).AddRow(
			"801", "M00000018", "织里晨星童装", "factory", "", "", "织里童装城", "一期", "1F", "A108",
			"织里童装城", `["童装"]`, 30.8732, 120.2255, time.Time{},
		))

	item, err := NewMerchantDirectoryModel(db).GetVisibleMerchantLocation(context.Background(), "801")
	if err != nil || item.MerchantID != "801" {
		t.Fatalf("item=%#v err=%v, want visible zhili merchant", item, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestBuildNearbyVisibleMerchantsQueryConservativelyContainsHighLatitudeRadius(t *testing.T) {
	const radiusMeters = int64(1000)
	origin := MerchantDirectoryItem{MerchantID: "801", Latitude: 89, Longitude: 120}
	angularRadius := float64(radiusMeters) / merchantDirectoryEarthRadiusMeters
	originLatitudeRadians := origin.Latitude * math.Pi / 180
	// 球面小圆在最大经差处的候选点；线性 angularRadius/cos(lat) 会在这里略微缩小 bbox。
	candidate := MerchantDirectoryItem{
		MerchantID: "802",
		Latitude:   math.Asin(math.Sin(originLatitudeRadians)/math.Cos(angularRadius)) * 180 / math.Pi,
		Longitude:  origin.Longitude + math.Asin(math.Sin(angularRadius)/math.Cos(originLatitudeRadians))*180/math.Pi,
	}
	if distance := haversineDistanceMeters(origin.Latitude, origin.Longitude, candidate.Latitude, candidate.Longitude); distance > float64(radiusMeters)+0.001 {
		t.Fatalf("fixture distance = %.9f, want inside %dm radius", distance, radiusMeters)
	}

	_, args := buildNearbyVisibleMerchantsQuery(origin, radiusMeters)
	if len(args) != 5 {
		t.Fatalf("args = %#v, want bounded longitude", args)
	}
	maxLongitude, ok := args[4].(float64)
	if !ok || maxLongitude < candidate.Longitude {
		t.Fatalf("maxLongitude = %.12f, candidate = %.12f; bbox must not exclude an in-radius high-latitude point", maxLongitude, candidate.Longitude)
	}
}

func TestListMerchantDirectoryRejectsOverflowingOffsetBeforeQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	_, err = NewMerchantDirectoryModel(db).ListMerchantDirectory(context.Background(), MerchantDirectoryFilter{
		CityCode: "zhili", Page: math.MaxInt64, PageSize: 100,
	})
	if err == nil || err.Error() != "商家目录分页参数超出范围" {
		t.Fatalf("error = %v, want pagination overflow rejection", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database must not be queried: %v", err)
	}
}

func TestGetVisibleMerchantLocationRejectsInvalidIDBeforeQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	_, err = NewMerchantDirectoryModel(db).GetVisibleMerchantLocation(context.Background(), "not-a-tsid")
	if err == nil || err.Error() != "商家编号格式不正确" {
		t.Fatalf("error = %v, want invalid merchant ID rejection", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database must not be queried: %v", err)
	}
}

func merchantDirectoryModelTestColumns() []string {
	return []string{
		"merchant_id", "merchant_no", "name", "merchant_type", "logo_url", "address_text",
		"market_name", "building_name", "floor_no", "booth_no", "location_name", "main_categories",
		"latitude", "longitude", "last_resource_active_at",
	}
}

func TestBuildNearbyVisibleMerchantsQueryHandlesDatelineAndPoles(t *testing.T) {
	datelineQuery, datelineArgs := buildNearbyVisibleMerchantsQuery(
		MerchantDirectoryItem{MerchantID: "801", Latitude: 0, Longitude: 179.999}, 1000,
	)
	if !strings.Contains(datelineQuery, "m.longitude >= $4 OR m.longitude <= $5") || len(datelineArgs) != 5 {
		t.Fatalf("dateline query = %q args=%#v, want wrapped longitude predicate", datelineQuery, datelineArgs)
	}

	polarQuery, polarArgs := buildNearbyVisibleMerchantsQuery(
		MerchantDirectoryItem{MerchantID: "802", Latitude: 89.999, Longitude: 10}, 1000,
	)
	if strings.Contains(polarQuery, "m.longitude BETWEEN $4") || strings.Contains(polarQuery, "m.longitude >= $4") {
		t.Fatalf("polar query = %q, longitude must use global range when bounding box reaches pole", polarQuery)
	}
	if len(polarArgs) != 3 {
		t.Fatalf("polar args = %#v, want merchant ID and latitude bounds", polarArgs)
	}
}

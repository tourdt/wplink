package model

import (
	"math"
	"strings"
	"testing"
)

func TestBuildMapObjectDerivedFieldsForRect(t *testing.T) {
	input := MapObjectInput{
		Code:         "A001",
		Name:         "A001 小鹿童装",
		Type:         "booth",
		GeometryType: "rect",
		Geometry:     JSONMap{"x": float64(520), "y": float64(260), "width": float64(80), "height": float64(50)},
		CategoryCodes: []string{
			"girl",
		},
		ServiceTags: []string{
			"spot",
		},
		Address: "利济路中段 A001",
	}

	fields, err := BuildMapObjectDerivedFields(input)
	if err != nil {
		t.Fatalf("BuildMapObjectDerivedFields() error = %v", err)
	}
	if fields.CenterX != 560 || fields.CenterY != 285 || fields.MinX != 520 || fields.MaxX != 600 {
		t.Fatalf("fields = %#v, want rect bounds", fields)
	}
	if !strings.Contains(fields.SearchText, "A001") || !strings.Contains(fields.SearchText, "小鹿童装") || !strings.Contains(fields.SearchText, "girl") {
		t.Fatalf("searchText = %q, want searchable code name category", fields.SearchText)
	}
}

func TestBuildMapObjectDerivedFieldsForPoint(t *testing.T) {
	input := MapObjectInput{
		Code:         "PACK_001",
		Name:         "利济路打包站",
		Type:         "packing_station",
		GeometryType: "point",
		Geometry:     JSONMap{"x": float64(860), "y": float64(420)},
	}

	fields, err := BuildMapObjectDerivedFields(input)
	if err != nil {
		t.Fatalf("BuildMapObjectDerivedFields() error = %v", err)
	}
	if fields.CenterX != 860 || fields.CenterY != 420 || fields.MinX != 860 || fields.MaxY != 420 {
		t.Fatalf("fields = %#v, want point bounds", fields)
	}
}

func TestBuildMapObjectDerivedFieldsForPolygon(t *testing.T) {
	input := MapObjectInput{
		Code:         "A088",
		Name:         "转角异形档口",
		Type:         "booth",
		GeometryType: "polygon",
		Geometry: JSONMap{"points": []interface{}{
			map[string]interface{}{"x": float64(100), "y": float64(120)},
			map[string]interface{}{"x": float64(220), "y": float64(120)},
			map[string]interface{}{"x": float64(240), "y": float64(180)},
			map[string]interface{}{"x": float64(130), "y": float64(210)},
		}},
	}

	fields, err := BuildMapObjectDerivedFields(input)
	if err != nil {
		t.Fatalf("BuildMapObjectDerivedFields() error = %v", err)
	}
	if fields.MinX != 100 || fields.MinY != 120 || fields.MaxX != 240 || fields.MaxY != 210 {
		t.Fatalf("fields = %#v, want polygon bounds", fields)
	}
	if fields.CenterX != 172.5 || fields.CenterY != 157.5 {
		t.Fatalf("fields = %#v, want polygon centroid average", fields)
	}
}

func TestBuildMapObjectDerivedFieldsRejectsInvalidPolygon(t *testing.T) {
	for _, geometry := range []JSONMap{
		{"points": []interface{}{
			map[string]interface{}{"x": float64(100), "y": float64(120)},
			map[string]interface{}{"x": float64(220), "y": float64(120)},
		}},
		{"points": []interface{}{
			map[string]interface{}{"x": float64(100), "y": float64(120)},
			map[string]interface{}{"x": float64(-1), "y": float64(120)},
			map[string]interface{}{"x": float64(220), "y": float64(180)},
		}},
	} {
		_, err := BuildMapObjectDerivedFields(MapObjectInput{
			Code:         "P001",
			Name:         "多边形档口",
			Type:         "booth",
			GeometryType: "polygon",
			Geometry:     geometry,
		})
		if err == nil {
			t.Fatalf("BuildMapObjectDerivedFields() error = nil, geometry=%#v", geometry)
		}
	}
}

func TestBuildMapObjectDerivedFieldsRejectsNonFiniteGeometryNumbers(t *testing.T) {
	cases := []struct {
		name         string
		geometryType string
		geometry     JSONMap
	}{
		{
			name:         "rect NaN x",
			geometryType: "rect",
			geometry:     JSONMap{"x": math.NaN(), "y": float64(120), "width": float64(80), "height": float64(50)},
		},
		{
			name:         "rect infinity width",
			geometryType: "rect",
			geometry:     JSONMap{"x": float64(100), "y": float64(120), "width": math.Inf(1), "height": float64(50)},
		},
		{
			name:         "polygon infinity vertex",
			geometryType: "polygon",
			geometry: JSONMap{"points": []interface{}{
				map[string]interface{}{"x": float64(100), "y": float64(120)},
				map[string]interface{}{"x": math.Inf(1), "y": float64(140)},
				map[string]interface{}{"x": float64(220), "y": float64(180)},
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildMapObjectDerivedFields(MapObjectInput{
				Code:         "A001",
				Name:         "异常坐标档口",
				Type:         "booth",
				GeometryType: tc.geometryType,
				Geometry:     tc.geometry,
			})
			if err == nil {
				t.Fatal("BuildMapObjectDerivedFields() error = nil, want non-finite geometry validation error")
			}
		})
	}
}

func TestSortNearbyMapObjectsOrdersByDistance(t *testing.T) {
	origin := MapObject{ID: "booth-1", CenterX: 100, CenterY: 100}
	candidates := []MapObject{
		{ID: "poi-far", Name: "远处物流", Type: "logistics_point", CenterX: 300, CenterY: 100},
		{ID: "poi-near", Name: "近处打包", Type: "packing_station", CenterX: 130, CenterY: 100},
	}

	items := SortNearbyMapObjects(origin, candidates, 1)
	if len(items) != 1 || items[0].ID != "poi-near" {
		t.Fatalf("items = %#v, want nearest poi", items)
	}
	if items[0].DistanceText != "30m" {
		t.Fatalf("distanceText = %q, want 30m", items[0].DistanceText)
	}
}

func TestBuildMerchantPlaceFilterSQLScopesPublishedBoothsAndDirectoryFilters(t *testing.T) {
	claimed := true
	whereSQL, args := buildMerchantPlaceFilterSQL(MerchantPlaceFilter{
		CityCode:      "zhili",
		Keyword:       "童装",
		Categories:    []string{"girl"},
		MerchantTypes: []string{"factory"},
		Claimed:       &claimed,
		Bounds: &GeoBoundsFilter{
			MinLat: 30.80,
			MaxLat: 30.90,
			MinLng: 120.20,
			MaxLng: 120.30,
		},
	})

	for _, token := range []string{
		"o.status = 'normal'",
		"o.layer = 'booth'",
		"s.status = 'published'",
		"city.code = $1",
		"o.merchant_id IS NOT NULL",
		"o.category_codes ?|",
		"m.merchant_type = ANY",
		"o.lat >=",
		"o.lng <=",
	} {
		if !strings.Contains(whereSQL, token) {
			t.Fatalf("whereSQL = %q, want token %q", whereSQL, token)
		}
	}
	if len(args) != 8 {
		t.Fatalf("args = %#v, want city keyword category type and four bounds", args)
	}
}

func TestBuildMerchantPlaceFilterSQLCanSelectPrelistedBooths(t *testing.T) {
	claimed := false
	whereSQL, _ := buildMerchantPlaceFilterSQL(MerchantPlaceFilter{Claimed: &claimed})

	if !strings.Contains(whereSQL, "o.merchant_id IS NULL") {
		t.Fatalf("whereSQL = %q, want prelisted filter", whereSQL)
	}
}

func TestRankNearbyMerchantPlacesFiltersRadiusOriginAndInvalidLocation(t *testing.T) {
	origin := MerchantPlace{Object: MapObject{ID: "object-1", MerchantID: "merchant-1", Lat: "30.8700000", Lng: "120.1200000"}}
	candidates := []MerchantPlace{
		{Object: MapObject{ID: "object-1", MerchantID: "merchant-1", Lat: "30.8700000", Lng: "120.1200000"}},
		{Object: MapObject{ID: "object-near", MerchantID: "merchant-2", Lat: "30.8705000", Lng: "120.1200000"}},
		{Object: MapObject{ID: "object-far", MerchantID: "merchant-3", Lat: "30.8900000", Lng: "120.1200000"}},
		{Object: MapObject{ID: "object-invalid", MerchantID: "merchant-4"}},
		{Object: MapObject{ID: "object-no-merchant", Lat: "30.8701000", Lng: "120.1200000"}},
	}

	items := rankNearbyMerchantPlaces(origin, candidates, 1000, 20)
	if len(items) != 1 || items[0].Object.MerchantID != "merchant-2" {
		t.Fatalf("items = %#v", items)
	}
	if items[0].DistanceMeters < 50 || items[0].DistanceMeters > 60 {
		t.Fatalf("distance = %d, want about 56m", items[0].DistanceMeters)
	}
}

func TestRankNearbyMerchantPlacesSortsByDistanceThenObjectIDAndLimits(t *testing.T) {
	origin := MerchantPlace{Object: MapObject{ID: "origin", MerchantID: "merchant-1", Lat: "30.8700000", Lng: "120.1200000"}}
	candidates := []MerchantPlace{
		{Object: MapObject{ID: "object-b", MerchantID: "merchant-3", Lat: "30.8705000", Lng: "120.1200000"}},
		{Object: MapObject{ID: "object-nearest", MerchantID: "merchant-4", Lat: "30.8701000", Lng: "120.1200000"}},
		{Object: MapObject{ID: "object-a", MerchantID: "merchant-2", Lat: "30.8705000", Lng: "120.1200000"}},
	}

	items := rankNearbyMerchantPlaces(origin, candidates, 1000, 2)
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Object.ID != "object-nearest" || items[1].Object.ID != "object-a" {
		t.Fatalf("item IDs = [%q, %q], want [object-nearest, object-a]", items[0].Object.ID, items[1].Object.ID)
	}
}

func TestBuildNearbyMerchantPlaceQueryScopesPublishedActiveBoothsInBoundingBox(t *testing.T) {
	origin := MerchantPlace{Object: MapObject{MerchantID: "merchant-1", Lat: "30.8700000", Lng: "120.1200000"}}
	query, args := buildNearbyMerchantPlaceQuery(origin, 1000)

	for _, token := range []string{
		"o.status = 'normal'",
		"o.layer = 'booth'",
		"s.status = 'published'",
		"m.id IS NOT NULL",
		"m.status = 'active'",
		"o.lat IS NOT NULL",
		"o.lng IS NOT NULL",
		"o.lat BETWEEN $2 AND $3",
		"o.lng BETWEEN $4 AND $5",
		"o.merchant_id <> $1::bigint",
	} {
		if !strings.Contains(query, token) {
			t.Fatalf("query = %q, want token %q", query, token)
		}
	}
	if strings.Contains(query, "o.phone") || strings.Contains(query, "o.wechat") {
		t.Fatalf("query = %q, must not read merchant contact fields", query)
	}
	if len(args) != 5 || args[0] != "merchant-1" {
		t.Fatalf("args = %#v, want merchant ID and four bounding values", args)
	}
	minLat, minLatOK := args[1].(float64)
	maxLat, maxLatOK := args[2].(float64)
	minLng, minLngOK := args[3].(float64)
	maxLng, maxLngOK := args[4].(float64)
	if !minLatOK || !maxLatOK || !minLngOK || !maxLngOK {
		t.Fatalf("bounding args = %#v, want float64 values", args[1:])
	}
	if minLat < 30.860 || minLat > 30.862 || maxLat < 30.878 || maxLat > 30.880 {
		t.Fatalf("latitude bounds = [%f, %f], want about 1km around 30.87", minLat, maxLat)
	}
	if minLng < 120.109 || minLng > 120.111 || maxLng < 120.129 || maxLng > 120.131 {
		t.Fatalf("longitude bounds = [%f, %f], want about 1km around 120.12", minLng, maxLng)
	}
}

func TestBuildNearbyMerchantPlaceQueryUsesGlobalLongitudeWhenRadiusReachesPole(t *testing.T) {
	for _, tc := range []struct {
		name string
		lat  string
	}{
		{name: "north pole", lat: "89.9950000"},
		{name: "south pole", lat: "-89.9950000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			origin := MerchantPlace{Object: MapObject{MerchantID: "merchant-1", Lat: tc.lat, Lng: "0.0000000"}}
			_, args := buildNearbyMerchantPlaceQuery(origin, 1000)
			if len(args) != 5 {
				t.Fatalf("args = %#v, want merchant ID and four bounding values", args)
			}
			minLng, minLngOK := args[3].(float64)
			maxLng, maxLngOK := args[4].(float64)
			if !minLngOK || !maxLngOK {
				t.Fatalf("longitude bounds = %#v, want float64 values", args[3:])
			}
			if minLng != -180 || maxLng != 180 {
				t.Fatalf("longitude bounds = [%f, %f], want global range for across-pole candidates", minLng, maxLng)
			}
		})
	}
}

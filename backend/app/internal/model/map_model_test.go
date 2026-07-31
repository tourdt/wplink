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

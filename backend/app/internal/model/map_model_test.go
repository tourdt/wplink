package model

import (
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

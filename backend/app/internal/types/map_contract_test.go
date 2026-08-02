package types

import (
	"reflect"
	"testing"
)

func TestMapObjectQueryTypesExposeViewportAndZoomFields(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target any
		fields map[string]string
	}{
		{
			name:   "public list objects",
			target: ListMapObjectsReq{},
			fields: map[string]string{
				"MinX": "form:\"minX,optional\"",
				"MinY": "form:\"minY,optional\"",
				"MaxX": "form:\"maxX,optional\"",
				"MaxY": "form:\"maxY,optional\"",
				"Zoom": "form:\"zoom,optional\"",
			},
		},
		{
			name:   "public search objects",
			target: SearchMapObjectsReq{},
			fields: map[string]string{
				"MinX": "form:\"minX,optional\"",
				"MinY": "form:\"minY,optional\"",
				"MaxX": "form:\"maxX,optional\"",
				"MaxY": "form:\"maxY,optional\"",
				"Zoom": "form:\"zoom,optional\"",
			},
		},
		{
			name:   "admin list objects",
			target: AdminListMapObjectsReq{},
			fields: map[string]string{
				"MinX": "form:\"minX,optional\"",
				"MinY": "form:\"minY,optional\"",
				"MaxX": "form:\"maxX,optional\"",
				"MaxY": "form:\"maxY,optional\"",
				"Zoom": "form:\"zoom,optional\"",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			targetType := reflect.TypeOf(tc.target)
			for fieldName, expectedTag := range tc.fields {
				field, ok := targetType.FieldByName(fieldName)
				if !ok {
					t.Fatalf("%s missing field %s", targetType.Name(), fieldName)
				}
				if string(field.Tag) != expectedTag {
					t.Fatalf("%s.%s tag = %q, want %q", targetType.Name(), fieldName, field.Tag, expectedTag)
				}
			}
		})
	}
}

func TestMerchantLocationContextTypesExposeCurrentNearbyAndDistance(t *testing.T) {
	respType := reflect.TypeOf(MerchantLocationContextResp{})
	for _, name := range []string{"Current", "Nearby", "RadiusMeters", "NearbyAvailable"} {
		if _, ok := respType.FieldByName(name); !ok {
			t.Fatalf("MerchantLocationContextResp missing %s", name)
		}
	}
	itemType := reflect.TypeOf(MerchantPlaceItem{})
	field, ok := itemType.FieldByName("DistanceMeters")
	if !ok || string(field.Tag) != `json:"distanceMeters,optional"` {
		t.Fatal("MerchantPlaceItem.DistanceMeters contract mismatch")
	}
}

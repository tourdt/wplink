package model

import (
	"strings"
	"testing"
)

func TestBusinessProfileNormalizesMultipleRolesAndSpecialtyItems(t *testing.T) {
	profile, err := NormalizeBusinessProfile(BusinessProfile{
		SchemaVersion: 1, EntityKind: "organization", PrimaryRole: "factory",
		Roles:             []string{"factory", "landlord", "service_provider"},
		ProcessingMethods: []string{" 车缝 ", "车缝", "印花"}, PropertyTypes: []string{"厂房"}, ServiceItems: []string{"打版"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.ProcessingMethods) != 2 || profile.ProcessingMethods[0] != "车缝" {
		t.Fatalf("methods = %#v", profile.ProcessingMethods)
	}
	if err := ValidateBusinessProfileComplete(profile, "多营主体", []string{"童装"}); err != nil {
		t.Fatal(err)
	}
	profile.ServiceItems = nil
	if err := ValidateBusinessProfileComplete(profile, "多营主体", []string{"童装"}); err == nil || !strings.Contains(err.Error(), "服务项目") {
		t.Fatalf("err = %v", err)
	}
}

func TestBusinessProfileRejectsInvalidVersionAndRoles(t *testing.T) {
	base := BusinessProfile{SchemaVersion: 1, EntityKind: "individual", PrimaryRole: "landlord", Roles: []string{"landlord"}, PropertyTypes: []string{"厂房"}}
	for _, tc := range []struct {
		name string
		edit func(*BusinessProfile)
	}{
		{"version", func(p *BusinessProfile) { p.SchemaVersion = 2 }},
		{"unknown role", func(p *BusinessProfile) { p.Roles = []string{"landlord", "buyer"} }},
		{"missing primary", func(p *BusinessProfile) { p.Roles = []string{"factory"} }},
		{"duplicate role", func(p *BusinessProfile) { p.Roles = []string{"landlord", "landlord"} }},
		{"missing entity kind", func(p *BusinessProfile) { p.EntityKind = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.edit(&p)
			if _, err := NormalizeBusinessProfile(p); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestLandlordCompletesWithoutCategoriesOrGPS(t *testing.T) {
	profile, err := NormalizeBusinessProfile(BusinessProfile{SchemaVersion: 1, EntityKind: "individual", PrimaryRole: "landlord", Roles: []string{"landlord"}, PropertyTypes: []string{"商铺"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBusinessProfileComplete(profile, "出租人", nil); err != nil {
		t.Fatal(err)
	}
	if profile.PrimaryRole != "landlord" {
		t.Fatal("primary role changed")
	}
}

func TestDecodeBusinessProfileRequiresKnownV1(t *testing.T) {
	profile, err := DecodeBusinessProfile([]byte(`{"schemaVersion":1,"entityKind":"","primaryRole":"","roles":[],"serviceArea":"","processingMethods":[],"propertyTypes":[],"serviceItems":[]}`))
	if err != nil || profile.PrimaryRole != "" || profile.SchemaVersion != 1 {
		t.Fatalf("profile=%#v err=%v", profile, err)
	}
	for _, raw := range []string{`{}`, `{"schemaVersion":2}`, `{"schemaVersion":1}`, `{"schemaVersion":1,"entityKind":"","primaryRole":"","roles":[],"serviceArea":"织里","processingMethods":[],"propertyTypes":[],"serviceItems":[]}`, `{"schemaVersion":1,"primaryRole":"unknown"}`} {
		if _, err := DecodeBusinessProfile([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

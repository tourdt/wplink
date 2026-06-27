package model

import (
	"database/sql/driver"
	"testing"
)

func TestJSONMapScansBytes(t *testing.T) {
	var value JSONMap

	if err := value.Scan([]byte(`{"season":"春款","allowLiveSale":true}`)); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if value["season"] != "春款" {
		t.Fatalf("season = %#v, want 春款", value["season"])
	}
	if value["allowLiveSale"] != true {
		t.Fatalf("allowLiveSale = %#v, want true", value["allowLiveSale"])
	}
}

func TestJSONMapValueReturnsJSONBytes(t *testing.T) {
	value := JSONMap{"typeCode": "inventory"}

	got, err := value.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	bytes, ok := got.([]byte)
	if !ok {
		t.Fatalf("Value() type = %T, want []byte", got)
	}
	if string(bytes) != `{"typeCode":"inventory"}` {
		t.Fatalf("Value() = %s, want compact JSON", string(bytes))
	}
	var _ driver.Valuer = value
}

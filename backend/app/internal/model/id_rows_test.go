package model

import "testing"

func TestPostgresTextIDRowsCastsIDColumns(t *testing.T) {
	got := postgresTextIDRows([]string{"id", "wechat_openid", "merchant_id", "created_by", "title"})
	want := "id::text AS id,wechat_openid,merchant_id::text AS merchant_id,created_by::text AS created_by,title"
	if got != want {
		t.Fatalf("postgresTextIDRows() = %q, want %q", got, want)
	}
}

package merchant

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/svc"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestListMerchantDirectoryHandlerParsesFiltersAndReturnsPublicFields(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	columns := merchantDirectoryTestColumns()
	mock.ExpectQuery(`(?s)SELECT.*FROM merchants m.*m\.main_categories \?\| \$3::text\[\].*ORDER BY.*LIMIT \$4 OFFSET \$5`).
		WithArgs("zhili", "A108", sqlmock.AnyArg(), int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(
			"801", "M00000018", "织里晨星童装", "factory", "logo.png", "利济路 88 号",
			"织里童装城", "一期", "1F", "A108", "织里童装城", `["童装"]`, 30.8732, 120.2255,
			time.Date(2026, 8, 28, 12, 30, 0, 0, time.FixedZone("CST", 8*60*60)),
		))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM merchants m.*m\.main_categories \?\| \$3::text\[\]`).
		WithArgs("zhili", "A108", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	svcCtx := &svc.ServiceContext{APIStore: &svc.APIStore{MerchantDirectoryModel: model.NewMerchantDirectoryModel(db)}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/merchant-directory?cityCode=zhili&keyword=A108&categories=%E7%AB%A5%E8%A3%85&page=1&pageSize=20", nil)
	recorder := httptest.NewRecorder()

	ListMerchantDirectoryHandler(svcCtx).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items, ok := envelope.Data["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("data = %#v, want one directory item", envelope.Data)
	}
	item, ok := items[0].(map[string]interface{})
	if !ok || item["merchantId"] != "801" || item["boothNo"] != "A108" {
		t.Fatalf("item = %#v, want merchant public location", items[0])
	}
	for _, forbidden := range []string{"contact", "contactName", "contactPhone", "contactWechat", "phone", "wechat"} {
		if _, exists := item[forbidden]; exists {
			t.Fatalf("item = %#v, public directory must not include %q", item, forbidden)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestGetMerchantLocationContextHandlerKeepsCurrentWhenNearbyQueryFails(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`(?s)SELECT.*FROM merchants m.*WHERE m\.id = \$1::bigint`).
		WithArgs("801").
		WillReturnRows(sqlmock.NewRows(merchantDirectoryTestColumns()).AddRow(
			"801", "M00000018", "织里晨星童装", "factory", "logo.png", "利济路 88 号",
			"织里童装城", "一期", "1F", "A108", "织里童装城", `["童装"]`, 30.8732, 120.2255, nil,
		))
	mock.ExpectQuery(`(?s)SELECT.*FROM merchants m.*m\.id <> \$1::bigint.*m\.latitude BETWEEN \$2 AND \$3`).
		WillReturnError(errors.New("nearby db unavailable"))

	svcCtx := &svc.ServiceContext{APIStore: &svc.APIStore{MerchantDirectoryModel: model.NewMerchantDirectoryModel(db)}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/801/location-context", nil)
	req = pathvar.WithVars(req, map[string]string{"merchantId": "801"})
	recorder := httptest.NewRecorder()

	GetMerchantLocationContextHandler(svcCtx).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"merchantId":"801"`) || !strings.Contains(body, `"nearbyAvailable":false`) || !strings.Contains(body, `"nearby":[]`) {
		t.Fatalf("body = %s, want current merchant and unavailable empty nearby list", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestGetMerchantLocationContextHandlerRejectsInvalidIDBeforeDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	svcCtx := &svc.ServiceContext{APIStore: &svc.APIStore{MerchantDirectoryModel: model.NewMerchantDirectoryModel(db)}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/not-a-tsid/location-context", nil)
	req = pathvar.WithVars(req, map[string]string{"merchantId": "not-a-tsid"})
	recorder := httptest.NewRecorder()

	GetMerchantLocationContextHandler(svcCtx).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"msg":"该商家位置待完善"`) {
		t.Fatalf("status = %d body=%s, want stable validation response", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database must not be queried: %v", err)
	}
}

func merchantDirectoryTestColumns() []string {
	return []string{
		"merchant_id", "merchant_no", "name", "merchant_type", "logo_url", "address_text",
		"market_name", "building_name", "floor_no", "booth_no", "location_name", "main_categories",
		"latitude", "longitude", "last_resource_active_at",
	}
}

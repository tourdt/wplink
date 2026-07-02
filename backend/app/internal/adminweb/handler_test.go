package adminweb

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHandlerServesStaticAssetUnderAdminBase(t *testing.T) {
	handler := NewHandler(fstest.MapFS{
		"index.html":         {Data: []byte("<html>admin</html>")},
		"assets/app-test.js": {Data: []byte("console.log('admin')")},
	}, "/admin/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/assets/app-test.js", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "console.log('admin')" {
		t.Fatalf("body = %q, want asset content", rec.Body.String())
	}
}

func TestHandlerFallsBackToIndexForAdminHistoryRoute(t *testing.T) {
	handler := NewHandler(fstest.MapFS{
		"index.html": {Data: []byte("<html>admin shell</html>")},
	}, "/admin/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/resources/pending", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "<html>admin shell</html>" {
		t.Fatalf("body = %q, want index fallback", rec.Body.String())
	}
}

func TestHandlerReturnsNotFoundForMissingAsset(t *testing.T) {
	handler := NewHandler(fstest.MapFS{
		"index.html": {Data: []byte("<html>admin shell</html>")},
	}, "/admin/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/assets/missing.js", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestEmbeddedAdminDistIncludesSourcingMapPage(t *testing.T) {
	var bundle strings.Builder
	err := fs.WalkDir(embeddedDist, "dist", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || (!strings.HasSuffix(name, ".html") && !strings.HasSuffix(name, ".js")) {
			return nil
		}

		data, err := embeddedDist.ReadFile(name)
		if err != nil {
			return err
		}
		bundle.Write(data)
		bundle.WriteByte('\n')
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded admin dist: %v", err)
	}

	source := bundle.String()
	for _, token := range []string{"sourcing-map", "拿货地图", "/api/v1/admin/map/scenes", "主营分类", "营业时间", "物流线路", "标准标签", "保存标签", "筛选标签", "筛选点位", "筛选场景", "默认视野", "设为当前画布中心", "批量生成点位", "factory_booth", "logistics_point", "点位排序", "最小显示级别", "最大显示级别", "定位", "状态操作", "object-status-badge", "搜索编码/名称", "全部状态"} {
		if !strings.Contains(source, token) {
			t.Fatalf("embedded admin dist missing %q", token)
		}
	}
}

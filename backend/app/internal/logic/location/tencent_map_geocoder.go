package location

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
	"wplink/backend/common/externalcall"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultTencentMapGeocoderURL        = "https://apis.map.qq.com/ws/geocoder/v1/"
	tencentMapStatusDailyQuotaExhausted = 121
)

type ReverseGeocodeReq struct {
	Latitude  string
	Longitude string
}

type ReverseGeocodeResp struct {
	Address  string `json:"address"`
	Name     string `json:"name,omitempty"`
	Province string `json:"province,omitempty"`
}

type ReverseGeocoder interface {
	ReverseGeocode(ctx context.Context, latitude float64, longitude float64) (ReverseGeocodeResp, error)
}

type Logic struct {
	geocoder ReverseGeocoder
}

func NewLogic(geocoder ReverseGeocoder) *Logic {
	return &Logic{geocoder: geocoder}
}

func (l *Logic) ReverseGeocode(ctx context.Context, req ReverseGeocodeReq) (ReverseGeocodeResp, error) {
	latitude, err := parseCoordinate(req.Latitude, -90, 90, "纬度")
	if err != nil {
		return ReverseGeocodeResp{}, err
	}
	longitude, err := parseCoordinate(req.Longitude, -180, 180, "经度")
	if err != nil {
		return ReverseGeocodeResp{}, err
	}
	if l == nil || l.geocoder == nil {
		logx.Errorf("逆地理编码服务未配置: latitude=%.6f longitude=%.6f", latitude, longitude)
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "地址解析服务未配置，请手动填写详细地址")
	}

	resp, err := l.geocoder.ReverseGeocode(ctx, latitude, longitude)
	if err != nil {
		return ReverseGeocodeResp{}, err
	}
	if strings.TrimSpace(resp.Address) == "" {
		logx.Errorf("逆地理编码未返回地址: latitude=%.6f longitude=%.6f", latitude, longitude)
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "未解析到详细地址，请搜索并选择具体地点")
	}
	resp.Address = strings.TrimSpace(resp.Address)
	resp.Name = strings.TrimSpace(resp.Name)
	resp.Province = strings.TrimSpace(resp.Province)
	return resp, nil
}

func parseCoordinate(raw string, min float64, max float64, label string) (float64, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return 0, errx.New(errx.CodeValidationFailed, "请提供"+label)
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || value < min || value > max {
		return 0, errx.New(errx.CodeValidationFailed, label+"不正确，请重新选择地图位置")
	}
	return value, nil
}

type TencentMapGeocoder struct {
	cfg      config.TencentMapConfig
	baseURL  string
	client   *http.Client
	observer externalcall.Observer
}

func NewTencentMapGeocoder(cfg config.TencentMapConfig, client *http.Client, observers ...externalcall.Observer) *TencentMapGeocoder {
	if strings.TrimSpace(cfg.Key) == "" {
		return nil
	}
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	// variadic 仅用于保持旧调用兼容：这是可选单 Observer，传入多个时只使用第一个。
	var observer externalcall.Observer
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &TencentMapGeocoder{
		cfg:      cfg,
		baseURL:  defaultTencentMapGeocoderURL,
		client:   client,
		observer: observer,
	}
}

func (g *TencentMapGeocoder) WithBaseURL(baseURL string) *TencentMapGeocoder {
	if strings.TrimSpace(baseURL) != "" {
		g.baseURL = strings.TrimSpace(baseURL)
	}
	return g
}

func (g *TencentMapGeocoder) ReverseGeocode(ctx context.Context, latitude float64, longitude float64) (ReverseGeocodeResp, error) {
	if g == nil || strings.TrimSpace(g.cfg.Key) == "" {
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "地址解析服务未配置，请手动填写详细地址")
	}
	endpoint, err := url.Parse(g.baseURL)
	if err != nil {
		// URL 配置可能带 query，日志只保留安全分类，避免凭据随原始地址或解析错误泄露。
		logx.Errorf("腾讯地图逆地理编码地址配置错误: category=url_parse")
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "地址解析服务配置错误，请手动填写详细地址")
	}
	query := endpoint.Query()
	query.Set("location", fmt.Sprintf("%.6f,%.6f", latitude, longitude))
	query.Set("key", g.cfg.Key)
	query.Set("get_poi", "1")
	endpoint.RawQuery = query.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		// 请求错误可能引用带 key 的完整 URL，因此不记录原始错误，只保留安全诊断上下文。
		logx.Errorf("创建腾讯地图逆地理编码请求失败: latitude=%.6f longitude=%.6f category=request_create", latitude, longitude)
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "地址解析失败，请手动填写详细地址")
	}
	// 请求创建成功才进入外呼观测，避免把本地 URL 配置错误误计为腾讯地图故障。
	call := externalcall.Start(g.observer, externalcall.ProviderTencentMap, externalcall.OperationReverseGeocode)
	httpResp, err := g.client.Do(httpReq)
	if err != nil {
		outcome := externalcall.ClassifyTransport(ctx, err)
		call.Finish(ctx, outcome, 0)
		// http.Client 会用 *url.Error 包装错误并附带含 key 的完整 URL，日志只能记录安全分类。
		logx.Errorf("腾讯地图逆地理编码请求失败: latitude=%.6f longitude=%.6f outcome=%s", latitude, longitude, outcome)
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "地址解析服务暂不可用，请手动填写详细地址")
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		call.Finish(ctx, externalcall.OutcomeTransportError, httpResp.StatusCode)
		logx.Errorf("读取腾讯地图逆地理编码响应失败: latitude=%.6f longitude=%.6f outcome=%s status=%d", latitude, longitude, externalcall.OutcomeTransportError, httpResp.StatusCode)
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "地址解析失败，请手动填写详细地址")
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		call.Finish(ctx, externalcall.OutcomeHTTPError, httpResp.StatusCode)
		logx.Errorf("腾讯地图逆地理编码 HTTP 状态异常: latitude=%.6f longitude=%.6f status=%d", latitude, longitude, httpResp.StatusCode)
		if httpResp.StatusCode == http.StatusTooManyRequests {
			return ReverseGeocodeResp{}, errx.New(errx.CodeRateLimited, "地址解析服务调用过于频繁，请手动填写详细地址")
		}
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "地址解析服务暂不可用，请手动填写详细地址")
	}

	var decoded tencentMapGeocoderResp
	if err := json.Unmarshal(body, &decoded); err != nil {
		call.Finish(ctx, externalcall.OutcomeDecodeError, httpResp.StatusCode)
		logx.Errorf("解析腾讯地图逆地理编码响应失败: latitude=%.6f longitude=%.6f err=%+v", latitude, longitude, err)
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "地址解析响应异常，请手动填写详细地址")
	}
	if decoded.Status != 0 {
		call.Finish(ctx, externalcall.OutcomeProviderRejected, httpResp.StatusCode)
		logx.Errorf("腾讯地图逆地理编码返回错误: latitude=%.6f longitude=%.6f status=%d message=%s", latitude, longitude, decoded.Status, decoded.Message)
		if isTencentMapQuotaExhausted(decoded.Status, decoded.Message) {
			return ReverseGeocodeResp{}, errx.New(errx.CodeRateLimited, "地址解析今日额度已用完，请手动填写详细地址")
		}
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "地址解析失败，请手动填写详细地址")
	}
	address := chooseTencentAddress(decoded.Result)
	if address == "" {
		call.Finish(ctx, externalcall.OutcomeDecodeError, httpResp.StatusCode)
		return ReverseGeocodeResp{}, errx.New(errx.CodeInternalError, "未解析到详细地址，请搜索并选择具体地点")
	}
	call.Finish(ctx, externalcall.OutcomeSuccess, httpResp.StatusCode)
	return ReverseGeocodeResp{
		Address:  address,
		Name:     chooseTencentPOIName(decoded.Result.POIs),
		Province: decoded.Result.AddressComponent.Province,
	}, nil
}

type tencentMapGeocoderResp struct {
	Status  int                      `json:"status"`
	Message string                   `json:"message"`
	Result  tencentMapGeocoderResult `json:"result"`
}

type tencentMapGeocoderResult struct {
	Address            string `json:"address"`
	FormattedAddresses struct {
		Recommend string `json:"recommend"`
		Rough     string `json:"rough"`
	} `json:"formatted_addresses"`
	AddressComponent struct {
		Province string `json:"province"`
	} `json:"address_component"`
	POIs []tencentMapPOI `json:"pois"`
}

type tencentMapPOI struct {
	Title   string `json:"title"`
	Address string `json:"address"`
}

func chooseTencentAddress(result tencentMapGeocoderResult) string {
	for _, candidate := range []string{result.FormattedAddresses.Recommend, result.Address, result.FormattedAddresses.Rough} {
		if text := strings.TrimSpace(candidate); text != "" {
			return text
		}
	}
	for _, poi := range result.POIs {
		if text := strings.TrimSpace(poi.Address); text != "" {
			return text
		}
	}
	return ""
}

func chooseTencentPOIName(pois []tencentMapPOI) string {
	for _, poi := range pois {
		if text := strings.TrimSpace(poi.Title); text != "" {
			return text
		}
	}
	return ""
}

func isTencentMapQuotaExhausted(status int, message string) bool {
	// 腾讯地图在 HTTP 200 内用业务状态码表达 Key 级别限制，避免误判为服务内部 500。
	if status == tencentMapStatusDailyQuotaExhausted {
		return true
	}
	return strings.Contains(message, "调用量已达到上限")
}

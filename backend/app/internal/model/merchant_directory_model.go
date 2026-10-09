package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

const merchantDirectoryEarthRadiusMeters = 6371000.0

type MerchantDirectoryFilter struct {
	CityCode   string
	Keyword    string
	Categories []string
	Page       int64
	PageSize   int64
}

// MerchantDirectoryItem 只承载目录公开字段，刻意不包含联系人、电话或微信。
type MerchantDirectoryItem struct {
	MerchantID           string
	MerchantNo           string
	Name                 string
	MerchantType         string
	LogoURL              string
	AddressText          string
	MarketName           string
	BuildingName         string
	FloorNo              string
	BoothNo              string
	LocationName         string
	MainCategories       []string
	Latitude             float64
	Longitude            float64
	LastResourceActiveAt time.Time
	DistanceMeters       float64
	PrimaryRole          string
	BusinessRoles        []string
}

type MerchantDirectoryModel struct {
	db *sql.DB
}

func NewMerchantDirectoryModel(db *sql.DB) *MerchantDirectoryModel {
	return &MerchantDirectoryModel{db: db}
}

const merchantDirectorySelectSQL = `SELECT
  m.id::text,
  COALESCE(m.merchant_no, ''),
  m.name,
  m.merchant_type,
  COALESCE(m.logo_url, ''),
  COALESCE(m.address_text, ''),
  COALESCE(m.market_name, ''),
  COALESCE(m.building_name, ''),
  COALESCE(m.floor_no, ''),
  COALESCE(m.booth_no, ''),
  COALESCE(m.location_name, ''),
  m.main_categories,
  m.latitude,
  m.longitude,
  latest_resource.last_resource_active_at,
  m.business_profile
FROM merchants m
JOIN city_stations cs ON cs.id = m.city_station_id
LEFT JOIN LATERAL (
  SELECT MAX(COALESCE(r.refreshed_at, r.published_at, r.created_at)) AS last_resource_active_at
  FROM resources r
  WHERE r.merchant_id = m.id
    AND r.status = 'published'
    AND r.deleted_at IS NULL
) latest_resource ON TRUE`

type merchantDirectoryWhere struct {
	SQL                   string
	Args                  []interface{}
	KeywordParameterIndex int
}

func buildMerchantDirectoryWhere(filter MerchantDirectoryFilter) (string, []interface{}) {
	built := buildMerchantDirectoryWhereDetails(filter)
	return built.SQL, built.Args
}

func buildMerchantDirectoryWhereDetails(filter MerchantDirectoryFilter) merchantDirectoryWhere {
	conditions := []string{
		"m.deleted_at IS NULL",
		"m.status = 'active'",
		"m.owner_visible = TRUE",
		"m.map_visible = TRUE",
		"m.profile_status = 'completed'",
		"m.location_status = 'visible'",
		"m.latitude IS NOT NULL",
		"m.longitude IS NOT NULL",
		"m.latitude BETWEEN -90 AND 90",
		"m.longitude BETWEEN -180 AND 180",
	}
	args := make([]interface{}, 0, 3)
	if cityCode := strings.TrimSpace(filter.CityCode); cityCode != "" {
		args = append(args, cityCode)
		conditions = append(conditions, fmt.Sprintf("cs.code = $%d", len(args)))
	}

	keywordParameterIndex := 0
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		args = append(args, keyword)
		keywordParameterIndex = len(args)
		placeholder := fmt.Sprintf("$%d", keywordParameterIndex)
		conditions = append(conditions, fmt.Sprintf(`(
  m.name ILIKE '%%' || %[1]s || '%%'
  OR m.market_name ILIKE '%%' || %[1]s || '%%'
  OR m.building_name ILIKE '%%' || %[1]s || '%%'
  OR m.floor_no ILIKE '%%' || %[1]s || '%%'
  OR m.booth_no ILIKE '%%' || %[1]s || '%%'
)`, placeholder))
	}

	categories := cleanDirectoryCategories(filter.Categories)
	if len(categories) > 0 {
		args = append(args, pq.Array(categories))
		conditions = append(conditions, fmt.Sprintf("m.main_categories ?| $%d::text[]", len(args)))
	}
	return merchantDirectoryWhere{
		SQL:                   " WHERE " + strings.Join(conditions, " AND "),
		Args:                  args,
		KeywordParameterIndex: keywordParameterIndex,
	}
}

func cleanDirectoryCategories(values []string) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

func (m *MerchantDirectoryModel) ListMerchantDirectory(ctx context.Context, filter MerchantDirectoryFilter) ([]MerchantDirectoryItem, error) {
	_, pageSize, offset, err := normalizeMerchantDirectoryPagination(filter.Page, filter.PageSize)
	if err != nil {
		return nil, err
	}
	built := buildMerchantDirectoryWhereDetails(filter)
	query := merchantDirectorySelectSQL + built.SQL + merchantDirectoryOrderSQL(built.KeywordParameterIndex)
	built.Args = append(built.Args, pageSize, offset)
	query += fmt.Sprintf("\nLIMIT $%d OFFSET $%d", len(built.Args)-1, len(built.Args))

	rows, err := m.db.QueryContext(ctx, query, built.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MerchantDirectoryItem, 0, pageSize)
	for rows.Next() {
		item, scanErr := scanMerchantDirectoryItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func normalizeMerchantDirectoryPagination(page int64, pageSize int64) (int64, int64, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	// 先检查再乘法，避免恶意超大页码绕成负 OFFSET 并被误报为数据库故障。
	if page > 1 && page-1 > math.MaxInt64/pageSize {
		return 0, 0, 0, errors.New("商家目录分页参数超出范围")
	}
	return page, pageSize, (page - 1) * pageSize, nil
}

func (m *MerchantDirectoryModel) CountMerchantDirectory(ctx context.Context, filter MerchantDirectoryFilter) (int64, error) {
	where, args := buildMerchantDirectoryWhere(filter)
	var total int64
	err := m.db.QueryRowContext(ctx, `SELECT COUNT(*)
FROM merchants m
JOIN city_stations cs ON cs.id = m.city_station_id`+where, args...).Scan(&total)
	return total, err
}

func merchantDirectoryOrderSQL(keywordParameterIndex int) string {
	keywordOrder := "0"
	if keywordParameterIndex > 0 {
		placeholder := fmt.Sprintf("$%d", keywordParameterIndex)
		keywordOrder = fmt.Sprintf(`CASE
  WHEN LOWER(COALESCE(m.booth_no, '')) = LOWER(%[1]s) THEN 0
  WHEN LOWER(m.name) = LOWER(%[1]s) THEN 1
  WHEN m.name ILIKE %[1]s || '%%'
    OR COALESCE(m.market_name, '') ILIKE %[1]s || '%%'
    OR COALESCE(m.building_name, '') ILIKE %[1]s || '%%'
    OR COALESCE(m.floor_no, '') ILIKE %[1]s || '%%'
    OR COALESCE(m.booth_no, '') ILIKE %[1]s || '%%' THEN 2
  ELSE 3
END`, placeholder)
	}
	return fmt.Sprintf(`
ORDER BY %s ASC,
  latest_resource.last_resource_active_at DESC NULLS LAST,
  m.last_active_at DESC NULLS LAST,
  m.id DESC`, keywordOrder)
}

func (m *MerchantDirectoryModel) GetVisibleMerchantLocation(ctx context.Context, merchantID string) (MerchantDirectoryItem, error) {
	merchantID = strings.TrimSpace(merchantID)
	if !IsValidMerchantDirectoryID(merchantID) {
		return MerchantDirectoryItem{}, errors.New("商家编号格式不正确")
	}
	query := merchantDirectorySelectSQL + ` WHERE m.id = $1::bigint
  AND cs.code = 'zhili'
  AND m.deleted_at IS NULL
  AND m.status = 'active'
  AND m.owner_visible = TRUE
  AND m.map_visible = TRUE
  AND m.profile_status = 'completed'
  AND m.location_status = 'visible'
  AND m.latitude IS NOT NULL
  AND m.longitude IS NOT NULL
  AND m.latitude BETWEEN -90 AND 90
  AND m.longitude BETWEEN -180 AND 180
LIMIT 1`
	return scanMerchantDirectoryItem(m.db.QueryRowContext(ctx, query, merchantID))
}

func (m *MerchantDirectoryModel) ListNearbyVisibleMerchants(ctx context.Context, origin MerchantDirectoryItem, radiusMeters int64, limit int64) ([]MerchantDirectoryItem, error) {
	query, args := buildNearbyVisibleMerchantsQuery(origin, radiusMeters)
	if query == "" || limit <= 0 {
		return []MerchantDirectoryItem{}, nil
	}
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]MerchantDirectoryItem, 0)
	for rows.Next() {
		item, scanErr := scanMerchantDirectoryItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rankNearbyVisibleMerchants(origin, candidates, radiusMeters, limit), nil
}

type merchantDirectoryScanner interface {
	Scan(dest ...interface{}) error
}

func scanMerchantDirectoryItem(scanner merchantDirectoryScanner) (MerchantDirectoryItem, error) {
	var item MerchantDirectoryItem
	var categories JSONStringSlice
	var lastResourceActiveAt sql.NullTime
	var businessProfileRaw []byte
	err := scanner.Scan(
		&item.MerchantID,
		&item.MerchantNo,
		&item.Name,
		&item.MerchantType,
		&item.LogoURL,
		&item.AddressText,
		&item.MarketName,
		&item.BuildingName,
		&item.FloorNo,
		&item.BoothNo,
		&item.LocationName,
		&categories,
		&item.Latitude,
		&item.Longitude,
		&lastResourceActiveAt,
		&businessProfileRaw,
	)
	if err != nil {
		return MerchantDirectoryItem{}, err
	}
	item.MainCategories = append([]string{}, categories...)
	profile, err := DecodeBusinessProfile(businessProfileRaw)
	if err != nil {
		return MerchantDirectoryItem{}, err
	}
	item.PrimaryRole = profile.PrimaryRole
	item.BusinessRoles = append([]string{}, profile.Roles...)
	if lastResourceActiveAt.Valid {
		item.LastResourceActiveAt = lastResourceActiveAt.Time
	}
	return item, nil
}

func buildNearbyVisibleMerchantsQuery(origin MerchantDirectoryItem, radiusMeters int64) (string, []interface{}) {
	merchantID := strings.TrimSpace(origin.MerchantID)
	if !IsValidMerchantDirectoryID(merchantID) || !validDirectoryCoordinates(origin.Latitude, origin.Longitude) || radiusMeters <= 0 {
		return "", nil
	}
	angularRadius := float64(radiusMeters) / merchantDirectoryEarthRadiusMeters
	latitudeDelta := angularRadius * 180 / math.Pi
	minLatitude := math.Max(-90, origin.Latitude-latitudeDelta)
	maxLatitude := math.Min(90, origin.Latitude+latitudeDelta)
	args := []interface{}{merchantID, minLatitude, maxLatitude}

	conditions := []string{
		"m.id <> $1::bigint",
		"cs.code = 'zhili'",
		"m.deleted_at IS NULL",
		"m.status = 'active'",
		"m.owner_visible = TRUE",
		"m.map_visible = TRUE",
		"m.profile_status = 'completed'",
		"m.location_status = 'visible'",
		"m.latitude IS NOT NULL",
		"m.longitude IS NOT NULL",
		"m.latitude BETWEEN -90 AND 90",
		"m.longitude BETWEEN -180 AND 180",
		"m.latitude BETWEEN $2 AND $3",
	}

	// 极区的外接矩形会覆盖全部经度；此时省略经度条件，防止漏掉跨极点的邻近商家。
	if minLatitude > -90 && maxLatitude < 90 {
		cosLatitude := math.Abs(math.Cos(origin.Latitude * math.Pi / 180))
		longitudeRatio := math.Abs(math.Sin(angularRadius)) / cosLatitude
		// asin 给出球面小圆的最大经差；定义域触边时使用全经度，宁可多取候选也不能漏掉半径内商家。
		if longitudeRatio < 1 {
			longitudeDelta := math.Asin(longitudeRatio) * 180 / math.Pi
			minimumLongitude := origin.Longitude - longitudeDelta
			maximumLongitude := origin.Longitude + longitudeDelta
			switch {
			case minimumLongitude < -180:
				minimumLongitude += 360
				args = append(args, minimumLongitude, maximumLongitude)
				conditions = append(conditions, "(m.longitude >= $4 OR m.longitude <= $5)")
			case maximumLongitude > 180:
				maximumLongitude -= 360
				args = append(args, minimumLongitude, maximumLongitude)
				conditions = append(conditions, "(m.longitude >= $4 OR m.longitude <= $5)")
			default:
				args = append(args, minimumLongitude, maximumLongitude)
				conditions = append(conditions, "m.longitude BETWEEN $4 AND $5")
			}
		}
	}
	return merchantDirectorySelectSQL + " WHERE " + strings.Join(conditions, " AND "), args
}

// IsValidMerchantDirectoryID 在任何 PostgreSQL bigint cast 前校验正整数 TSID，避免把用户参数错误记录成数据库故障。
func IsValidMerchantDirectoryID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return err == nil && parsed > 0
}

func rankNearbyVisibleMerchants(origin MerchantDirectoryItem, candidates []MerchantDirectoryItem, radiusMeters int64, limit int64) []MerchantDirectoryItem {
	if strings.TrimSpace(origin.MerchantID) == "" || !validDirectoryCoordinates(origin.Latitude, origin.Longitude) || radiusMeters <= 0 || limit <= 0 {
		return []MerchantDirectoryItem{}
	}
	items := make([]MerchantDirectoryItem, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.MerchantID) == "" || candidate.MerchantID == origin.MerchantID || !validDirectoryCoordinates(candidate.Latitude, candidate.Longitude) {
			continue
		}
		distance := haversineDistanceMeters(origin.Latitude, origin.Longitude, candidate.Latitude, candidate.Longitude)
		if distance > float64(radiusMeters) {
			continue
		}
		candidate.DistanceMeters = distance
		items = append(items, candidate)
	}
	sort.SliceStable(items, func(left, right int) bool {
		if math.Abs(items[left].DistanceMeters-items[right].DistanceMeters) > 1e-9 {
			return items[left].DistanceMeters < items[right].DistanceMeters
		}
		return items[left].MerchantID < items[right].MerchantID
	})
	if int64(len(items)) > limit {
		items = items[:limit]
	}
	return items
}

func validDirectoryCoordinates(latitude float64, longitude float64) bool {
	return ValidMerchantCoordinates(latitude, longitude)
}

func haversineDistanceMeters(originLatitude float64, originLongitude float64, targetLatitude float64, targetLongitude float64) float64 {
	originLatitudeRadians := originLatitude * math.Pi / 180
	targetLatitudeRadians := targetLatitude * math.Pi / 180
	latitudeDelta := (targetLatitude - originLatitude) * math.Pi / 180
	longitudeDelta := (targetLongitude - originLongitude) * math.Pi / 180
	a := math.Sin(latitudeDelta/2)*math.Sin(latitudeDelta/2) +
		math.Cos(originLatitudeRadians)*math.Cos(targetLatitudeRadians)*math.Sin(longitudeDelta/2)*math.Sin(longitudeDelta/2)
	return merchantDirectoryEarthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

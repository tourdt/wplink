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
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const (
	MapSceneStatusDraft     = "draft"
	MapSceneStatusPublished = "published"
	MapSceneStatusArchived  = "archived"

	MapObjectStatusNormal = "normal"
	MapObjectStatusHidden = "hidden"
	MapObjectStatusClosed = "closed"

	MapCategoryStatusNormal = "normal"
	MapCategoryStatusHidden = "hidden"
	MapCategoryStatusClosed = "closed"

	MapGeometryTypeRect  = "rect"
	MapGeometryTypePoint = "point"
)

type MapScene struct {
	ID             string
	CityCode       string
	Code           string
	Name           string
	Type           string
	ParentCode     string
	BackgroundURL  string
	Width          int64
	Height         int64
	MinScale       string
	MaxScale       string
	DefaultScale   string
	DefaultCenterX string
	DefaultCenterY string
	FloorNo        string
	Sort           int64
	Revision       int64
	Status         string
	CreatedAt      string
	UpdatedAt      string
}

type MapSceneInput struct {
	CityCode       string
	Code           string
	Name           string
	Type           string
	ParentCode     string
	BackgroundURL  string
	Width          int64
	Height         int64
	MinScale       string
	MaxScale       string
	DefaultScale   string
	DefaultCenterX string
	DefaultCenterY string
	FloorNo        string
	Sort           int64
	Status         string
}

type MapObject struct {
	ID             string
	SceneCode      string
	MerchantID     string
	Code           string
	Name           string
	Type           string
	Layer          string
	GeometryType   string
	Geometry       JSONMap
	CenterX        float64
	CenterY        float64
	MinX           float64
	MinY           float64
	MaxX           float64
	MaxY           float64
	MinZoom        int64
	MaxZoom        int64
	CategoryCodes  []string
	ServiceTags    []string
	PlatformTags   []string
	PoiServiceTags []string
	Address        string
	Phone          string
	Wechat         string
	Lat            string
	Lng            string
	SearchText     string
	Extra          JSONMap
	Sort           int64
	Status         string
	DistanceText   string
	CreatedAt      string
	UpdatedAt      string
}

type MapObjectInput struct {
	ID             string
	SceneCode      string
	MerchantID     string
	Code           string
	Name           string
	Type           string
	Layer          string
	GeometryType   string
	Geometry       JSONMap
	MinZoom        int64
	MaxZoom        int64
	CategoryCodes  []string
	ServiceTags    []string
	PlatformTags   []string
	PoiServiceTags []string
	Address        string
	Phone          string
	Wechat         string
	Lat            string
	Lng            string
	Extra          JSONMap
	Sort           int64
	Status         string
}

type MapObjectDerivedFields struct {
	CenterX    float64
	CenterY    float64
	MinX       float64
	MinY       float64
	MaxX       float64
	MaxY       float64
	SearchText string
}

type MapCategory struct {
	ID        string
	Code      string
	Name      string
	Type      string
	IconURL   string
	Sort      int64
	IsVisible bool
	Status    string
	CreatedAt string
	UpdatedAt string
}

type MapCategoryInput struct {
	Code      string
	Name      string
	Type      string
	IconURL   string
	Sort      int64
	IsVisible bool
	Status    string
}

type ListMapCategoriesFilter struct {
	Type   string
	Status string
}

type ListMapScenesFilter struct {
	CityCode   string
	ParentCode string
	Type       string
	Status     string
}

type ListMapObjectsFilter struct {
	SceneCode      string
	Types          []string
	Categories     []string
	ServiceTags    []string
	PoiServiceTags []string
	Keyword        string
	Status         string
	Limit          int64
}

type MapModel struct {
	db   *sql.DB
	conn sqlx.SqlConn
}

func NewMapModel(db *sql.DB) *MapModel {
	return &MapModel{db: db, conn: sqlx.NewSqlConnFromDB(db)}
}

func BuildMapObjectDerivedFields(input MapObjectInput) (MapObjectDerivedFields, error) {
	geometryType := strings.TrimSpace(input.GeometryType)
	var centerX, centerY, minX, minY, maxX, maxY float64

	switch geometryType {
	case MapGeometryTypeRect:
		x, err := numberFromGeometry(input.Geometry, "x")
		if err != nil {
			return MapObjectDerivedFields{}, err
		}
		y, err := numberFromGeometry(input.Geometry, "y")
		if err != nil {
			return MapObjectDerivedFields{}, err
		}
		width, err := numberFromGeometry(input.Geometry, "width")
		if err != nil {
			return MapObjectDerivedFields{}, err
		}
		height, err := numberFromGeometry(input.Geometry, "height")
		if err != nil {
			return MapObjectDerivedFields{}, err
		}
		if width <= 0 || height <= 0 {
			return MapObjectDerivedFields{}, errors.New("地图标注尺寸必须大于 0")
		}
		centerX = x + width/2
		centerY = y + height/2
		minX = x
		minY = y
		maxX = x + width
		maxY = y + height
	case MapGeometryTypePoint:
		x, err := numberFromGeometry(input.Geometry, "x")
		if err != nil {
			return MapObjectDerivedFields{}, err
		}
		y, err := numberFromGeometry(input.Geometry, "y")
		if err != nil {
			return MapObjectDerivedFields{}, err
		}
		centerX, centerY = x, y
		minX, minY, maxX, maxY = x, y, x, y
	default:
		return MapObjectDerivedFields{}, errors.New("地图标注形状不支持")
	}

	return MapObjectDerivedFields{
		CenterX:    centerX,
		CenterY:    centerY,
		MinX:       minX,
		MinY:       minY,
		MaxX:       maxX,
		MaxY:       maxY,
		SearchText: buildMapObjectSearchText(input),
	}, nil
}

type NearbyMapObject struct {
	ID           string
	Name         string
	Type         string
	DistanceText string
	CenterX      float64
	CenterY      float64
}

func SortNearbyMapObjects(origin MapObject, candidates []MapObject, limit int) []NearbyMapObject {
	type rankedObject struct {
		object   MapObject
		distance float64
	}

	ranked := make([]rankedObject, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ID == origin.ID {
			continue
		}
		distance := math.Hypot(candidate.CenterX-origin.CenterX, candidate.CenterY-origin.CenterY)
		ranked = append(ranked, rankedObject{object: candidate, distance: distance})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].distance < ranked[j].distance
	})

	if limit <= 0 || limit > len(ranked) {
		limit = len(ranked)
	}
	items := make([]NearbyMapObject, 0, limit)
	for _, item := range ranked[:limit] {
		items = append(items, NearbyMapObject{
			ID:           item.object.ID,
			Name:         item.object.Name,
			Type:         item.object.Type,
			DistanceText: formatMapDistance(item.distance),
			CenterX:      item.object.CenterX,
			CenterY:      item.object.CenterY,
		})
	}
	return items
}

func (m *MapModel) ListPublishedScenes(ctx context.Context, filter ListMapScenesFilter) ([]MapScene, error) {
	filter.Status = MapSceneStatusPublished
	return m.ListAdminScenes(ctx, filter)
}

func (m *MapModel) GetPublishedScene(ctx context.Context, sceneCode string) (MapScene, error) {
	return m.getScene(ctx, strings.TrimSpace(sceneCode), MapSceneStatusPublished)
}

func (m *MapModel) GetAdminScene(ctx context.Context, sceneCode string) (MapScene, error) {
	return m.getScene(ctx, strings.TrimSpace(sceneCode), "")
}

func (m *MapModel) ListPublishedObjects(ctx context.Context, filter ListMapObjectsFilter) ([]MapObject, error) {
	filter.Status = MapObjectStatusNormal
	return m.listObjects(ctx, filter)
}

func (m *MapModel) SearchPublishedObjects(ctx context.Context, filter ListMapObjectsFilter) ([]MapObject, error) {
	filter.Status = MapObjectStatusNormal
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	return m.listObjects(ctx, filter)
}

func (m *MapModel) GetPublishedObject(ctx context.Context, objectID string) (MapObject, error) {
	return m.getObject(ctx, strings.TrimSpace(objectID), MapObjectStatusNormal)
}

func (m *MapModel) ListObjectsBySceneAndTypes(ctx context.Context, sceneCode string, types []string) ([]MapObject, error) {
	return m.listObjects(ctx, ListMapObjectsFilter{
		SceneCode: strings.TrimSpace(sceneCode),
		Types:     cleanStringSlice(types),
		Status:    MapObjectStatusNormal,
	})
}

func (m *MapModel) ListAdminObjects(ctx context.Context, filter ListMapObjectsFilter) ([]MapObject, error) {
	return m.listObjects(ctx, filter)
}

func (m *MapModel) ListAdminScenes(ctx context.Context, filter ListMapScenesFilter) ([]MapScene, error) {
	query := `
SELECT s.id::text, COALESCE(c.code, ''), s.code, s.name, s.type, COALESCE(s.parent_code, ''),
       s.background_url, s.width, s.height, s.min_scale::text, s.max_scale::text,
       s.default_scale::text, COALESCE(s.default_center_x::text, ''), COALESCE(s.default_center_y::text, ''),
       COALESCE(s.floor_no, ''), s.sort::bigint, s.revision::bigint, s.status,
       s.created_at, s.updated_at
FROM map_scene s
LEFT JOIN city_stations c ON c.id = s.city_station_id
`
	conditions := make([]string, 0, 4)
	args := make([]interface{}, 0, 4)
	if v := strings.TrimSpace(filter.CityCode); v != "" {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("c.code = $%d", len(args)))
	}
	if v := strings.TrimSpace(filter.ParentCode); v != "" {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("s.parent_code = $%d", len(args)))
	}
	if v := strings.TrimSpace(filter.Type); v != "" {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("s.type = $%d", len(args)))
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("s.status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY s.sort ASC, s.created_at ASC"

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scenes := make([]MapScene, 0)
	for rows.Next() {
		scene, err := scanMapScene(rows)
		if err != nil {
			return nil, err
		}
		scenes = append(scenes, scene)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return scenes, nil
}

func (m *MapModel) SaveScene(ctx context.Context, input MapSceneInput) (MapScene, error) {
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = MapSceneStatusDraft
	}
	row := m.db.QueryRowContext(ctx, `
INSERT INTO map_scene (
  city_station_id, code, name, type, parent_code, background_url, width, height,
  min_scale, max_scale, default_scale, default_center_x, default_center_y,
  floor_no, sort, status, updated_at
) VALUES (
  CASE WHEN $1 = '' THEN NULL ELSE (SELECT id FROM city_stations WHERE code = $1 LIMIT 1) END,
  $2, $3, $4, NULLIF($5, ''), $6, $7, $8,
  COALESCE(NULLIF($9, '')::numeric, 0.5), COALESCE(NULLIF($10, '')::numeric, 5),
  COALESCE(NULLIF($11, '')::numeric, 1), NULLIF($12, '')::numeric, NULLIF($13, '')::numeric,
  NULLIF($14, ''), $15, $16, now()
) ON CONFLICT (code) DO UPDATE SET
  city_station_id = EXCLUDED.city_station_id,
  name = EXCLUDED.name,
  type = EXCLUDED.type,
  parent_code = EXCLUDED.parent_code,
  background_url = EXCLUDED.background_url,
  width = EXCLUDED.width,
  height = EXCLUDED.height,
  min_scale = EXCLUDED.min_scale,
  max_scale = EXCLUDED.max_scale,
  default_scale = EXCLUDED.default_scale,
  default_center_x = EXCLUDED.default_center_x,
  default_center_y = EXCLUDED.default_center_y,
  floor_no = EXCLUDED.floor_no,
  sort = EXCLUDED.sort,
  status = EXCLUDED.status,
  revision = map_scene.revision + 1,
  updated_at = now()
RETURNING id::text, COALESCE((SELECT code FROM city_stations WHERE id = map_scene.city_station_id), ''),
          code, name, type, COALESCE(parent_code, ''), background_url, width, height,
          min_scale::text, max_scale::text, default_scale::text,
          COALESCE(default_center_x::text, ''), COALESCE(default_center_y::text, ''),
          COALESCE(floor_no, ''), sort::bigint, revision::bigint, status, created_at, updated_at
`, strings.TrimSpace(input.CityCode), strings.TrimSpace(input.Code), strings.TrimSpace(input.Name), strings.TrimSpace(input.Type),
		strings.TrimSpace(input.ParentCode), strings.TrimSpace(input.BackgroundURL), input.Width, input.Height,
		strings.TrimSpace(input.MinScale), strings.TrimSpace(input.MaxScale), strings.TrimSpace(input.DefaultScale),
		strings.TrimSpace(input.DefaultCenterX), strings.TrimSpace(input.DefaultCenterY), strings.TrimSpace(input.FloorNo),
		input.Sort, status)
	return scanMapScene(row)
}

func (m *MapModel) PublishScene(ctx context.Context, sceneCode string) (MapScene, error) {
	row := m.db.QueryRowContext(ctx, `
UPDATE map_scene
SET status = 'published', revision = revision + 1, updated_at = now()
WHERE code = $1
RETURNING id::text, COALESCE((SELECT code FROM city_stations WHERE id = map_scene.city_station_id), ''),
          code, name, type, COALESCE(parent_code, ''), background_url, width, height,
          min_scale::text, max_scale::text, default_scale::text,
          COALESCE(default_center_x::text, ''), COALESCE(default_center_y::text, ''),
          COALESCE(floor_no, ''), sort::bigint, revision::bigint, status, created_at, updated_at
`, strings.TrimSpace(sceneCode))
	return scanMapScene(row)
}

func (m *MapModel) SaveObject(ctx context.Context, input MapObjectInput) (MapObject, error) {
	derived, err := BuildMapObjectDerivedFields(input)
	if err != nil {
		return MapObject{}, err
	}
	if input.MinZoom <= 0 {
		input.MinZoom = 1
	}
	if input.MaxZoom <= 0 {
		input.MaxZoom = 5
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = MapObjectStatusNormal
	}

	if strings.TrimSpace(input.ID) != "" {
		row := m.db.QueryRowContext(ctx, `
UPDATE map_object
SET code = $2, name = $3, type = $4, layer = $5, geometry_type = $6, geometry = $7,
    center_x = $8, center_y = $9, min_x = $10, min_y = $11, max_x = $12, max_y = $13,
    min_zoom = $14, max_zoom = $15, category_codes = $16, service_tags = $17,
    platform_tags = $18, poi_service_tags = $19, address = NULLIF($20, ''), phone = NULLIF($21, ''),
    wechat = NULLIF($22, ''), lat = NULLIF($23, '')::numeric, lng = NULLIF($24, '')::numeric,
    search_text = $25, extra = $26, sort = $27, status = $28, updated_at = now()
WHERE id::text = $1
RETURNING `+mapObjectSelectColumns(),
			strings.TrimSpace(input.ID), strings.TrimSpace(input.Code), strings.TrimSpace(input.Name), strings.TrimSpace(input.Type),
			strings.TrimSpace(input.Layer), strings.TrimSpace(input.GeometryType), input.Geometry,
			derived.CenterX, derived.CenterY, derived.MinX, derived.MinY, derived.MaxX, derived.MaxY,
			input.MinZoom, input.MaxZoom, JSONStringSlice(cleanStringSlice(input.CategoryCodes)), JSONStringSlice(cleanStringSlice(input.ServiceTags)),
			JSONStringSlice(cleanStringSlice(input.PlatformTags)), JSONStringSlice(cleanStringSlice(input.PoiServiceTags)),
			strings.TrimSpace(input.Address), strings.TrimSpace(input.Phone), strings.TrimSpace(input.Wechat),
			strings.TrimSpace(input.Lat), strings.TrimSpace(input.Lng), derived.SearchText, nonNilJSONMap(input.Extra), input.Sort, status)
		return scanMapObject(row)
	}

	row := m.db.QueryRowContext(ctx, `
INSERT INTO map_object (
  scene_code, merchant_id, code, name, type, layer, geometry_type, geometry,
  center_x, center_y, min_x, min_y, max_x, max_y, min_zoom, max_zoom,
  category_codes, service_tags, platform_tags, poi_service_tags,
  address, phone, wechat, lat, lng, search_text, extra, sort, status, updated_at
) VALUES (
  $1, CASE WHEN $2 = '' THEN NULL ELSE $2::bigint END, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13, $14, $15, $16,
  $17, $18, $19, $20,
  NULLIF($21, ''), NULLIF($22, ''), NULLIF($23, ''), NULLIF($24, '')::numeric, NULLIF($25, '')::numeric,
  $26, $27, $28, $29, now()
) ON CONFLICT (scene_code, code) DO UPDATE SET
  merchant_id = EXCLUDED.merchant_id,
  name = EXCLUDED.name,
  type = EXCLUDED.type,
  layer = EXCLUDED.layer,
  geometry_type = EXCLUDED.geometry_type,
  geometry = EXCLUDED.geometry,
  center_x = EXCLUDED.center_x,
  center_y = EXCLUDED.center_y,
  min_x = EXCLUDED.min_x,
  min_y = EXCLUDED.min_y,
  max_x = EXCLUDED.max_x,
  max_y = EXCLUDED.max_y,
  min_zoom = EXCLUDED.min_zoom,
  max_zoom = EXCLUDED.max_zoom,
  category_codes = EXCLUDED.category_codes,
  service_tags = EXCLUDED.service_tags,
  platform_tags = EXCLUDED.platform_tags,
  poi_service_tags = EXCLUDED.poi_service_tags,
  address = EXCLUDED.address,
  phone = EXCLUDED.phone,
  wechat = EXCLUDED.wechat,
  lat = EXCLUDED.lat,
  lng = EXCLUDED.lng,
  search_text = EXCLUDED.search_text,
  extra = EXCLUDED.extra,
  sort = EXCLUDED.sort,
  status = EXCLUDED.status,
  updated_at = now()
RETURNING `+mapObjectSelectColumns(),
		strings.TrimSpace(input.SceneCode), strings.TrimSpace(input.MerchantID), strings.TrimSpace(input.Code),
		strings.TrimSpace(input.Name), strings.TrimSpace(input.Type), strings.TrimSpace(input.Layer), strings.TrimSpace(input.GeometryType),
		input.Geometry, derived.CenterX, derived.CenterY, derived.MinX, derived.MinY, derived.MaxX, derived.MaxY,
		input.MinZoom, input.MaxZoom, JSONStringSlice(cleanStringSlice(input.CategoryCodes)), JSONStringSlice(cleanStringSlice(input.ServiceTags)),
		JSONStringSlice(cleanStringSlice(input.PlatformTags)), JSONStringSlice(cleanStringSlice(input.PoiServiceTags)),
		strings.TrimSpace(input.Address), strings.TrimSpace(input.Phone), strings.TrimSpace(input.Wechat),
		strings.TrimSpace(input.Lat), strings.TrimSpace(input.Lng), derived.SearchText, nonNilJSONMap(input.Extra), input.Sort, status)
	return scanMapObject(row)
}

func (m *MapModel) UpdateObjectStatus(ctx context.Context, objectID string, status string) (MapObject, error) {
	row := m.db.QueryRowContext(ctx, `
UPDATE map_object
SET status = $2, updated_at = now()
WHERE id::text = $1
RETURNING `+mapObjectSelectColumns(), strings.TrimSpace(objectID), strings.TrimSpace(status))
	return scanMapObject(row)
}

func (m *MapModel) BatchCreateObjects(ctx context.Context, inputs []MapObjectInput) ([]MapObject, error) {
	objects := make([]MapObject, 0, len(inputs))
	for _, input := range inputs {
		object, err := m.SaveObject(ctx, input)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func (m *MapModel) ListCategories(ctx context.Context, filter ListMapCategoriesFilter) ([]MapCategory, error) {
	query := `
SELECT id::text, code, name, type, COALESCE(icon_url, ''), sort::bigint, is_visible, status, created_at, updated_at
FROM map_category
`
	args := make([]interface{}, 0, 2)
	conditions := make([]string, 0, 2)
	if v := strings.TrimSpace(filter.Type); v != "" {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)))
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY sort ASC, created_at ASC"

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MapCategory, 0)
	for rows.Next() {
		item, err := scanMapCategory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *MapModel) SaveCategory(ctx context.Context, input MapCategoryInput) (MapCategory, error) {
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = MapCategoryStatusNormal
	}
	row := m.db.QueryRowContext(ctx, `
INSERT INTO map_category(code, name, type, icon_url, sort, is_visible, status, updated_at)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, now())
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  type = EXCLUDED.type,
  icon_url = EXCLUDED.icon_url,
  sort = EXCLUDED.sort,
  is_visible = EXCLUDED.is_visible,
  status = EXCLUDED.status,
  updated_at = now()
RETURNING id::text, code, name, type, COALESCE(icon_url, ''), sort::bigint, is_visible, status, created_at, updated_at
`, strings.TrimSpace(input.Code), strings.TrimSpace(input.Name), strings.TrimSpace(input.Type), strings.TrimSpace(input.IconURL), input.Sort, input.IsVisible, status)
	return scanMapCategory(row)
}

func (m *MapModel) getScene(ctx context.Context, sceneCode string, status string) (MapScene, error) {
	row := m.db.QueryRowContext(ctx, `
SELECT s.id::text, COALESCE(c.code, ''), s.code, s.name, s.type, COALESCE(s.parent_code, ''),
       s.background_url, s.width, s.height, s.min_scale::text, s.max_scale::text,
       s.default_scale::text, COALESCE(s.default_center_x::text, ''), COALESCE(s.default_center_y::text, ''),
       COALESCE(s.floor_no, ''), s.sort::bigint, s.revision::bigint, s.status,
       s.created_at, s.updated_at
FROM map_scene s
LEFT JOIN city_stations c ON c.id = s.city_station_id
WHERE s.code = $1 AND ($2 = '' OR s.status = $2)
LIMIT 1
`, sceneCode, strings.TrimSpace(status))
	return scanMapScene(row)
}

func (m *MapModel) getObject(ctx context.Context, objectID string, status string) (MapObject, error) {
	row := m.db.QueryRowContext(ctx, `
SELECT `+mapObjectSelectColumns()+`
FROM map_object
WHERE id::text = $1 AND ($2 = '' OR status = $2)
LIMIT 1
`, objectID, strings.TrimSpace(status))
	return scanMapObject(row)
}

func (m *MapModel) listObjects(ctx context.Context, filter ListMapObjectsFilter) ([]MapObject, error) {
	query := `SELECT ` + mapObjectSelectColumns() + ` FROM map_object`
	conditions := make([]string, 0, 8)
	args := make([]interface{}, 0, 8)
	if v := strings.TrimSpace(filter.SceneCode); v != "" {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("scene_code = $%d", len(args)))
	}
	if len(filter.Types) > 0 {
		args = append(args, pq.Array(cleanStringSlice(filter.Types)))
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", len(args)))
	}
	if len(filter.Categories) > 0 {
		args = append(args, pq.Array(cleanStringSlice(filter.Categories)))
		conditions = append(conditions, fmt.Sprintf("category_codes ?| $%d", len(args)))
	}
	if len(filter.ServiceTags) > 0 {
		args = append(args, pq.Array(cleanStringSlice(filter.ServiceTags)))
		conditions = append(conditions, fmt.Sprintf("service_tags ?| $%d", len(args)))
	}
	if len(filter.PoiServiceTags) > 0 {
		args = append(args, pq.Array(cleanStringSlice(filter.PoiServiceTags)))
		conditions = append(conditions, fmt.Sprintf("poi_service_tags ?| $%d", len(args)))
	}
	if v := strings.TrimSpace(filter.Keyword); v != "" {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("(code ILIKE '%%' || $%d || '%%' OR name ILIKE '%%' || $%d || '%%' OR search_text ILIKE '%%' || $%d || '%%')", len(args), len(args), len(args)))
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		args = append(args, v)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY sort ASC, code ASC"
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	objects := make([]MapObject, 0)
	for rows.Next() {
		item, err := scanMapObject(rows)
		if err != nil {
			return nil, err
		}
		objects = append(objects, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return objects, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanMapScene(row rowScanner) (MapScene, error) {
	var scene MapScene
	var createdAt time.Time
	var updatedAt time.Time
	err := row.Scan(
		&scene.ID, &scene.CityCode, &scene.Code, &scene.Name, &scene.Type, &scene.ParentCode,
		&scene.BackgroundURL, &scene.Width, &scene.Height, &scene.MinScale, &scene.MaxScale,
		&scene.DefaultScale, &scene.DefaultCenterX, &scene.DefaultCenterY, &scene.FloorNo,
		&scene.Sort, &scene.Revision, &scene.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return MapScene{}, err
	}
	scene.CreatedAt = createdAt.Format(time.RFC3339)
	scene.UpdatedAt = updatedAt.Format(time.RFC3339)
	return scene, nil
}

func scanMapObject(row rowScanner) (MapObject, error) {
	var object MapObject
	var categoryCodes JSONStringSlice
	var serviceTags JSONStringSlice
	var platformTags JSONStringSlice
	var poiServiceTags JSONStringSlice
	var createdAt time.Time
	var updatedAt time.Time
	err := row.Scan(
		&object.ID, &object.SceneCode, &object.MerchantID, &object.Code, &object.Name,
		&object.Type, &object.Layer, &object.GeometryType, &object.Geometry,
		&object.CenterX, &object.CenterY, &object.MinX, &object.MinY, &object.MaxX, &object.MaxY,
		&object.MinZoom, &object.MaxZoom, &categoryCodes, &serviceTags, &platformTags, &poiServiceTags,
		&object.Address, &object.Phone, &object.Wechat, &object.Lat, &object.Lng,
		&object.SearchText, &object.Extra, &object.Sort, &object.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return MapObject{}, err
	}
	object.CategoryCodes = []string(categoryCodes)
	object.ServiceTags = []string(serviceTags)
	object.PlatformTags = []string(platformTags)
	object.PoiServiceTags = []string(poiServiceTags)
	object.CreatedAt = createdAt.Format(time.RFC3339)
	object.UpdatedAt = updatedAt.Format(time.RFC3339)
	return object, nil
}

func scanMapCategory(row rowScanner) (MapCategory, error) {
	var category MapCategory
	var createdAt time.Time
	var updatedAt time.Time
	err := row.Scan(&category.ID, &category.Code, &category.Name, &category.Type, &category.IconURL, &category.Sort, &category.IsVisible, &category.Status, &createdAt, &updatedAt)
	if err != nil {
		return MapCategory{}, err
	}
	category.CreatedAt = createdAt.Format(time.RFC3339)
	category.UpdatedAt = updatedAt.Format(time.RFC3339)
	return category, nil
}

func mapObjectSelectColumns() string {
	return `id::text, scene_code, COALESCE(merchant_id::text, ''), code, name, type, layer, geometry_type, geometry,
       COALESCE(center_x, 0)::float8, COALESCE(center_y, 0)::float8,
       COALESCE(min_x, 0)::float8, COALESCE(min_y, 0)::float8,
       COALESCE(max_x, 0)::float8, COALESCE(max_y, 0)::float8,
       min_zoom::bigint, max_zoom::bigint,
       category_codes, service_tags, platform_tags, poi_service_tags,
       COALESCE(address, ''), COALESCE(phone, ''), COALESCE(wechat, ''),
       COALESCE(lat::text, ''), COALESCE(lng::text, ''),
       search_text, extra, sort::bigint, status, created_at, updated_at`
}

func numberFromGeometry(geometry JSONMap, key string) (float64, error) {
	raw, ok := geometry[key]
	if !ok {
		return 0, fmt.Errorf("地图标注缺少 %s 坐标", key)
	}
	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case jsonNumber:
		return strconv.ParseFloat(string(v), 64)
	case string:
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	default:
		return 0, fmt.Errorf("地图标注 %s 坐标格式不正确", key)
	}
}

type jsonNumber string

func buildMapObjectSearchText(input MapObjectInput) string {
	parts := []string{
		strings.TrimSpace(input.Code),
		strings.TrimSpace(input.Name),
		strings.TrimSpace(input.Type),
		strings.TrimSpace(input.Layer),
		strings.TrimSpace(input.Address),
		strings.TrimSpace(input.Phone),
		strings.TrimSpace(input.Wechat),
	}
	parts = append(parts, cleanStringSlice(input.CategoryCodes)...)
	parts = append(parts, cleanStringSlice(input.ServiceTags)...)
	parts = append(parts, cleanStringSlice(input.PlatformTags)...)
	parts = append(parts, cleanStringSlice(input.PoiServiceTags)...)
	return strings.Join(nonEmptyStrings(parts), " ")
}

func cleanStringSlice(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func nonEmptyStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func nonNilJSONMap(value JSONMap) JSONMap {
	if value == nil {
		return JSONMap{}
	}
	return value
}

func formatMapDistance(distance float64) string {
	rounded := int(math.Round(distance))
	if rounded < 1000 {
		return fmt.Sprintf("%dm", rounded)
	}
	return fmt.Sprintf("%.1fkm", float64(rounded)/1000)
}

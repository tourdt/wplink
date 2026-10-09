package model

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

const (
	MerchantStatusActive            = "active"
	MerchantProfileStatusIncomplete = "incomplete"
	MerchantProfileStatusCompleted  = "completed"
	homeRecentMerchantsMaxLimit     = int64(6)
)

type CreditTag struct {
	Code  string
	Label string
}

type MerchantDetail struct {
	ID              string
	MerchantNo      string
	Name            string
	MerchantType    string
	CityCode        string
	MainCategories  []string
	ProfileStatus   string
	VIPStatus       string
	CreditTags      []CreditTag
	ContactName     string
	ContactPhone    string
	ContactWechat   string
	OwnerVisible    bool
	PublicPhone     bool
	PublicWechat    bool
	MapVisible      bool
	BusinessProfile BusinessProfile
	PhoneMasked     string
	WechatMasked    string
	PublishedCount  int64
	DealtCount      int64
	FollowerCount   int64
	AddressText     string
	Location        MerchantLocation
	Description     string
	LogoURL         string
	Images          []string
	LastActiveAt    string
}

type UpdateMerchantPatch struct {
	Name            string
	MainCategories  []string
	Description     string
	LogoURL         string
	Images          []string
	ContactName     string
	ContactPhone    string
	ContactWechat   string
	PublicPhone     *bool
	PublicWechat    *bool
	MapVisible      *bool
	BusinessProfile *BusinessProfile
	AddressText     string
	Location        MerchantLocationPatch
}

type ListMerchantsFilter struct {
	CityCode       string
	MerchantType   string
	Status         string
	LocationStatus string
	Keyword        string
	Page           int64
	PageSize       int64
}

type MerchantListItem struct {
	ID                         string
	Name                       string
	MerchantType               string
	Status                     string
	LocationStatus             string
	MarketName                 string
	BuildingName               string
	FloorNo                    string
	BoothNo                    string
	PendingLocationReportCount int64
	LastActiveAt               string
}

type HomeRecentMerchant struct {
	ID             string
	Name           string
	MerchantType   string
	MainCategories []string
	LogoURL        string
	AddressText    string
	OnboardedAt    string
}

type ListMerchantsResult struct {
	Items    []MerchantListItem
	Page     int64
	PageSize int64
	Total    int64
}

type MerchantModel struct {
	db *sql.DB
}

func NewMerchantModel(db *sql.DB) *MerchantModel {
	return &MerchantModel{db: db}
}

func (m *MerchantModel) GetMerchantDetail(ctx context.Context, merchantID string) (MerchantDetail, error) {
	var detail MerchantDetail
	var categories JSONStringSlice
	var images JSONStringSlice
	var lastActive sql.NullTime
	var marketName, buildingName, floorNo, boothNo, locationName sql.NullString
	var latitude, longitude sql.NullFloat64
	var locationUpdatedAt sql.NullTime
	var businessProfileRaw []byte

	err := m.db.QueryRowContext(ctx, `
SELECT
  m.id::text,
  COALESCE(m.merchant_no, ''),
  m.name,
  m.merchant_type,
  cs.code,
  m.main_categories,
  COALESCE(NULLIF(m.profile_status, ''), 'completed'),
  CASE WHEN EXISTS (
    SELECT 1
    FROM merchant_vip_subscriptions mvs
    WHERE mvs.merchant_id = m.id
      AND mvs.status = 'active'
      AND mvs.starts_at <= now()
      AND mvs.expires_at > now()
  ) THEN 'active' ELSE 'none' END AS vip_status,
  m.contact_name,
  m.contact_phone,
  COALESCE(m.contact_wechat, ''),
  m.owner_visible,
  m.public_phone,
  m.public_wechat,
  m.map_visible,
  m.business_profile,
  COALESCE(m.address_text, ''),
	COALESCE(m.market_name, ''),
	COALESCE(m.building_name, ''),
	COALESCE(m.floor_no, ''),
	COALESCE(m.booth_no, ''),
	m.latitude,
	m.longitude,
	COALESCE(m.location_name, ''),
	m.location_status,
	m.location_updated_at,
  COALESCE(m.description, ''),
  COALESCE(m.logo_url, ''),
  m.images,
  m.last_active_at,
  COUNT(r.id) FILTER (
    WHERE r.status = 'published'
      AND (r.expires_at IS NULL OR r.expires_at > now())
      AND (r.dealt_at IS NULL OR r.dealt_at > now() - interval '7 days')
  ) AS published_count,
  COUNT(r.id) FILTER (WHERE r.dealt_at IS NOT NULL) AS dealt_count,
  (
    SELECT COUNT(*)
    FROM user_followed_merchants ufm
    WHERE ufm.merchant_id = m.id AND ufm.status = 'active'
  ) AS follower_count
FROM merchants m
JOIN city_stations cs ON cs.id = m.city_station_id
LEFT JOIN resources r ON r.merchant_id = m.id AND r.deleted_at IS NULL
WHERE m.id = $1 AND m.deleted_at IS NULL
GROUP BY m.id, cs.code
`, merchantID).Scan(
		&detail.ID,
		&detail.MerchantNo,
		&detail.Name,
		&detail.MerchantType,
		&detail.CityCode,
		&categories,
		&detail.ProfileStatus,
		&detail.VIPStatus,
		&detail.ContactName,
		&detail.ContactPhone,
		&detail.ContactWechat,
		&detail.OwnerVisible,
		&detail.PublicPhone,
		&detail.PublicWechat,
		&detail.MapVisible,
		&businessProfileRaw,
		&detail.AddressText,
		&marketName,
		&buildingName,
		&floorNo,
		&boothNo,
		&latitude,
		&longitude,
		&locationName,
		&detail.Location.Status,
		&locationUpdatedAt,
		&detail.Description,
		&detail.LogoURL,
		&images,
		&lastActive,
		&detail.PublishedCount,
		&detail.DealtCount,
		&detail.FollowerCount,
	)
	if err != nil {
		return MerchantDetail{}, err
	}
	detail.BusinessProfile, err = DecodeBusinessProfile(businessProfileRaw)
	if err != nil {
		return MerchantDetail{}, err
	}

	detail.MainCategories = []string(categories)
	detail.ProfileStatus = normalizeMerchantProfileStatus(detail.ProfileStatus)
	detail.Images = []string(images)
	detail.Location.MarketName = marketName.String
	detail.Location.BuildingName = buildingName.String
	detail.Location.FloorNo = floorNo.String
	detail.Location.BoothNo = boothNo.String
	detail.Location.LocationName = locationName.String
	if latitude.Valid {
		detail.Location.Latitude = float64Pointer(latitude.Float64)
	}
	if longitude.Valid {
		detail.Location.Longitude = float64Pointer(longitude.Float64)
	}
	if locationUpdatedAt.Valid {
		detail.Location.UpdatedAt = locationUpdatedAt.Time.Format(time.RFC3339)
	}
	detail.PhoneMasked = maskContact(detail.ContactPhone)
	detail.WechatMasked = maskWechat(detail.ContactWechat)
	if lastActive.Valid {
		detail.LastActiveAt = lastActive.Time.Format(time.RFC3339)
	}

	tags, err := m.listMerchantCreditTags(ctx, merchantID)
	if err != nil {
		return MerchantDetail{}, err
	}
	detail.CreditTags = tags
	return detail, nil
}

func (m *MerchantModel) GetMerchantContactPhone(ctx context.Context, merchantID string) (string, error) {
	var phone string
	err := m.db.QueryRowContext(ctx, `
SELECT contact_phone
FROM merchants
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1
`, merchantID).Scan(&phone)
	return strings.TrimSpace(phone), err
}

func float64Pointer(value float64) *float64 {
	return &value
}

const listHomeRecentMerchantsSQL = `
SELECT
  m.id::text,
  m.name,
  m.merchant_type,
  m.main_categories,
  COALESCE(m.logo_url, ''),
  COALESCE(m.address_text, ''),
  m.onboarded_at
FROM merchants m
JOIN city_stations cs ON cs.id = m.city_station_id
WHERE m.status = 'active'
  AND m.profile_status = 'completed'
  AND m.deleted_at IS NULL
  AND m.onboarded_at IS NOT NULL
  AND cs.code = $1
  AND EXISTS (
    SELECT 1
    FROM merchant_admin_bindings mab
    WHERE mab.merchant_id = m.id
      AND mab.status = 'active'
      AND mab.role = 'owner'
  )
ORDER BY m.onboarded_at DESC, m.id DESC
LIMIT $2
`

func (m *MerchantModel) ListHomeRecentMerchants(ctx context.Context, cityCode string, limit int64) ([]HomeRecentMerchant, error) {
	if limit <= 0 || limit > homeRecentMerchantsMaxLimit {
		limit = homeRecentMerchantsMaxLimit
	}
	rows, err := m.db.QueryContext(ctx, listHomeRecentMerchantsSQL, strings.TrimSpace(cityCode), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]HomeRecentMerchant, 0, limit)
	for rows.Next() {
		var item HomeRecentMerchant
		var categories JSONStringSlice
		var onboardedAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.MerchantType,
			&categories,
			&item.LogoURL,
			&item.AddressText,
			&onboardedAt,
		); err != nil {
			return nil, err
		}
		item.MainCategories = []string(categories)
		item.OnboardedAt = onboardedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *MerchantModel) ListMerchants(ctx context.Context, filter ListMerchantsFilter) (ListMerchantsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	rows, err := m.db.QueryContext(ctx, `
SELECT
  m.id::text,
  m.name,
  m.merchant_type,
  m.status,
	m.location_status,
	COALESCE(m.market_name, ''),
	COALESCE(m.building_name, ''),
	COALESCE(m.floor_no, ''),
	COALESCE(m.booth_no, ''),
  m.last_active_at,
	(
		SELECT COUNT(*)
		FROM merchant_location_reports mlr
		WHERE mlr.merchant_id = m.id AND mlr.status = 'pending'
	) AS pending_location_report_count,
  COUNT(*) OVER() AS total
FROM merchants m
JOIN city_stations cs ON cs.id = m.city_station_id
WHERE m.deleted_at IS NULL
	  AND ($1 = '' OR cs.code = $1)
	  AND ($2 = '' OR m.merchant_type = $2)
	  AND ($3 = '' OR m.status = $3)
	  AND ($4 = '' OR m.location_status = $4)
	  AND ($5 = '' OR m.name ILIKE '%' || $5 || '%' OR m.contact_name ILIKE '%' || $5 || '%')
	ORDER BY m.created_at DESC
	LIMIT $6 OFFSET $7
	`, filter.CityCode, filter.MerchantType, filter.Status, filter.LocationStatus, filter.Keyword, pageSize, offset)
	if err != nil {
		return ListMerchantsResult{}, err
	}
	defer rows.Close()

	var items []MerchantListItem
	var total int64
	for rows.Next() {
		var item MerchantListItem
		var lastActive sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.Name, &item.MerchantType, &item.Status, &item.LocationStatus,
			&item.MarketName, &item.BuildingName, &item.FloorNo, &item.BoothNo, &lastActive,
			&item.PendingLocationReportCount, &total,
		); err != nil {
			return ListMerchantsResult{}, err
		}
		if lastActive.Valid {
			item.LastActiveAt = lastActive.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListMerchantsResult{}, err
	}

	return ListMerchantsResult{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (m *MerchantModel) listMerchantCreditTags(ctx context.Context, merchantID string) ([]CreditTag, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT tag_code, tag_label
FROM credit_records
WHERE merchant_id = $1
  AND visibility = 'public'
  AND revoked_at IS NULL
ORDER BY created_at DESC
`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []CreditTag
	for rows.Next() {
		var tag CreditTag
		if err := rows.Scan(&tag.Code, &tag.Label); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func normalizePage(page int64, pageSize int64) (int64, int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func maskContact(value string) string {
	if len(value) < 7 {
		return value
	}
	return value[:3] + "****" + value[len(value)-4:]
}

func maskWechat(value string) string {
	if len(value) <= 4 {
		return value
	}
	return value[:4] + "_****"
}

package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

func (m *MerchantModel) UpdateMerchant(ctx context.Context, merchantID string, patch UpdateMerchantPatch) (string, error) {
	startedAt := time.Now()
	updatedAt := startedAt.UTC()
	var oldLocationStatus, oldProfileStatus, newProfileStatus, oldMerchantType, nextMerchantType, oldAddress string
	var oldMapVisible bool
	var newMapVisible bool
	var oldBusinessProfileRaw []byte
	var oldLatitude, oldLongitude sql.NullFloat64
	var newBusinessProfileRaw []byte
	var newLocationStatus string
	var resolvedReportCount int64
	stage := "begin"
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		if patch.BusinessProfile == nil || patch.MapVisible == nil {
			return businessProfileError("请完善业务身份并选择地图展示设置")
		}
		var oldName string
		stage = "lock_merchant"
		if err := tx.QueryRowContext(ctx, `
SELECT
	merchant_type,
	location_status,
 name,
 profile_status,
 business_profile,
 map_visible,
 COALESCE(address_text, ''),
 latitude,
 longitude
FROM merchants
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE
`, merchantID).Scan(&oldMerchantType, &oldLocationStatus, &oldName, &oldProfileStatus, &oldBusinessProfileRaw, &oldMapVisible, &oldAddress, &oldLatitude, &oldLongitude); err != nil {
			return err
		}

		nextMerchantType = oldMerchantType
		effectiveName := oldName
		if patch.Name != "" {
			effectiveName = patch.Name
		}
		if _, err := DecodeBusinessProfile(oldBusinessProfileRaw); err != nil {
			return err
		}
		stage = "validate_profile"
		profile, validationErr := NormalizeBusinessProfile(*patch.BusinessProfile)
		if validationErr != nil {
			return validationErr
		}
		if validationErr := ValidateBusinessProfileComplete(profile, effectiveName, patch.MainCategories); validationErr != nil {
			return validationErr
		}
		newProfileStatus = MerchantProfileStatusCompleted
		nextMerchantType = profile.PrimaryRole
		encodedProfile, marshalErr := json.Marshal(profile)
		if marshalErr != nil {
			return marshalErr
		}
		newBusinessProfileRaw = encodedProfile
		merchantTypeChanged := nextMerchantType != oldMerchantType
		newMapVisible = *patch.MapVisible
		newLocationStatus = oldLocationStatus
		if patch.Location.Action == MerchantLocationActionSet {
			newLocationStatus = MerchantLocationStatusVisible
		}
		if patch.Location.Action == MerchantLocationActionClear {
			newLocationStatus = MerchantLocationStatusHidden
		}
		if *patch.MapVisible {
			stage = "validate_map"
			// 显式上地图只能使用已保存或本次新选的有效接待地点；风险隐藏的旧点位不会因开关恢复公开。
			latitude, longitude := oldLatitude.Float64, oldLongitude.Float64
			coordinatesAvailable := oldLatitude.Valid && oldLongitude.Valid
			if patch.Location.Action == MerchantLocationActionSet {
				latitude, longitude, coordinatesAvailable = patch.Location.Latitude, patch.Location.Longitude, true
			} else if patch.Location.Action == MerchantLocationActionClear {
				coordinatesAvailable = false
			}
			address := oldAddress
			if patch.Location.Action == MerchantLocationActionSet && patch.AddressText != "" {
				address = patch.AddressText
			}
			if !coordinatesAvailable || !ValidMerchantCoordinates(latitude, longitude) || strings.TrimSpace(address) == "" {
				return businessProfileError("请先完善详细地址并选择地图位置，再开启地图展示")
			}
		}

		stage = "save_profile"
		result, err := tx.ExecContext(ctx, updateMerchantSQL,
			merchantID,
			patch.Name,
			JSONStringSlice(patch.MainCategories),
			nextMerchantType,
			patch.Description,
			patch.LogoURL,
			JSONStringSlice(patch.Images),
			updatedAt,
			patch.ContactName,
			patch.ContactPhone,
			patch.ContactWechat,
			patch.AddressText,
			patch.Location.Action,
			patch.Location.MarketName,
			patch.Location.BuildingName,
			patch.Location.FloorNo,
			patch.Location.BoothNo,
			patch.Location.Latitude,
			patch.Location.Longitude,
			patch.Location.LocationName,
			patch.Location.Status,
			newProfileStatus,
			patch.PublicPhone,
			patch.PublicWechat,
			newBusinessProfileRaw,
			patch.MapVisible,
		)
		if err != nil {
			return err
		}
		stage = "affected_rows"
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return sql.ErrNoRows
		}
		if patch.Location.Action == MerchantLocationActionSet {
			// 商家重新选择有效点位即完成纠错；报告结案必须与新坐标、visible 状态同一提交，避免短暂公开旧风险状态。
			stage = "resolve_reports"
			resolvedReportCount, err = resolvePendingLocationReportsTx(ctx, tx, merchantID)
			if err != nil {
				return err
			}
		}

		if !merchantTypeChanged {
			stage = "commit"
			return nil
		}

		stage = "record_type_change"
		_, err = tx.ExecContext(ctx, `
INSERT INTO merchant_type_change_logs (
  merchant_id,
  old_merchant_type,
  new_merchant_type,
  changed_at
)
VALUES ($1, $2, $3, $4)
`, merchantID, oldMerchantType, nextMerchantType, updatedAt)
		if err != nil {
			return err
		}
		stage = "commit"
		return nil
	})
	if err != nil {
		return "", &MerchantOperationError{Stage: stage, Cause: err, DurationMs: time.Since(startedAt).Milliseconds()}
	}
	// 只在 Commit 成功后记录成功，避免报告结案或身份日志失败时出现“已更新”的假日志。
	logx.WithContext(ctx).Infow("商家资料事务已提交",
		logx.Field("operation", "merchant_profile_update"), logx.Field("merchantId", merchantID),
		logx.Field("locationAction", patch.Location.Action), logx.Field("stage", "committed"),
		logx.Field("oldLocationStatus", oldLocationStatus), logx.Field("newLocationStatus", newLocationStatus),
		logx.Field("oldMapVisible", oldMapVisible), logx.Field("newMapVisible", newMapVisible),
		logx.Field("schemaVersion", effectiveSchemaVersion(patch.BusinessProfile)), logx.Field("primaryRole", effectivePrimaryRole(patch.BusinessProfile)),
		logx.Field("oldProfileStatus", oldProfileStatus), logx.Field("newProfileStatus", newProfileStatus),
		logx.Field("merchantTypeChanged", oldMerchantType != nextMerchantType),
		logx.Field("resolvedReportCount", resolvedReportCount), logx.Field("durationMs", time.Since(startedAt).Milliseconds()),
	)
	return updatedAt.Format(time.RFC3339), nil
}

const updateMerchantSQL = `
UPDATE merchants
SET
  name = COALESCE(NULLIF($2, ''), name),
  main_categories = $3,
  merchant_type = $4,
  description = $5,
  logo_url = $6,
  images = $7,
  updated_at = $8,
  contact_name = COALESCE(NULLIF($9, ''), contact_name),
  contact_phone = COALESCE(NULLIF($10, ''), contact_phone),
  contact_wechat = COALESCE(NULLIF($11, ''), contact_wechat),
  address_text = CASE WHEN $13 = 'keep' THEN address_text ELSE COALESCE(NULLIF($12, ''), address_text) END,
	market_name = CASE WHEN $13 = 'set' THEN NULLIF($14, '') WHEN $13 = 'clear' THEN NULL ELSE market_name END,
	building_name = CASE WHEN $13 = 'set' THEN NULLIF($15, '') WHEN $13 = 'clear' THEN NULL ELSE building_name END,
	floor_no = CASE WHEN $13 = 'set' THEN NULLIF($16, '') WHEN $13 = 'clear' THEN NULL ELSE floor_no END,
	booth_no = CASE WHEN $13 = 'set' THEN NULLIF($17, '') WHEN $13 = 'clear' THEN NULL ELSE booth_no END,
	latitude = CASE WHEN $13 = 'set' THEN $18 WHEN $13 = 'clear' THEN NULL ELSE latitude END,
	longitude = CASE WHEN $13 = 'set' THEN $19 WHEN $13 = 'clear' THEN NULL ELSE longitude END,
	location_name = CASE WHEN $13 = 'set' THEN NULLIF($20, '') WHEN $13 = 'clear' THEN NULL ELSE location_name END,
	location_status = CASE WHEN $13 = 'set' THEN $21 WHEN $13 = 'clear' THEN 'hidden' ELSE location_status END,
	location_updated_at = CASE WHEN $13 = 'set' THEN $8 WHEN $13 = 'clear' THEN NULL ELSE location_updated_at END,
  onboarded_at = CASE WHEN $22 = 'completed' THEN COALESCE(onboarded_at, $8) ELSE onboarded_at END,
  profile_status = $22,
  public_phone = COALESCE($23, public_phone),
  public_wechat = COALESCE($24, public_wechat),
  business_profile = $25::jsonb,
  map_visible = $26
WHERE id = $1 AND deleted_at IS NULL
`

func effectiveSchemaVersion(profile *BusinessProfile) int {
	if profile == nil {
		return 0
	}
	return profile.SchemaVersion
}
func effectivePrimaryRole(profile *BusinessProfile) string {
	if profile == nil {
		return ""
	}
	return profile.PrimaryRole
}

package merchant

import (
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

// buildMerchantProfilePatch 只负责请求校验与归一化，不读取数据库，也不决定地图公开资格。
// 完整性和状态转换必须在持久化层持有商家行锁后计算，避免并发请求使用旧状态。
func buildMerchantProfilePatch(req UpdateMerchantReq) (model.UpdateMerchantPatch, string, error) {
	name := strings.TrimSpace(req.Name)
	if err := validateMerchantName(name); err != nil {
		return model.UpdateMerchantPatch{}, "name", err
	}
	phone := strings.TrimSpace(req.ContactPhone)
	if phone != "" && !isValidContactPhone(phone) {
		return model.UpdateMerchantPatch{}, "contact_phone", errx.New(errx.CodeValidationFailed, "手机号格式不正确")
	}
	if req.BusinessProfile == nil || req.MapVisible == nil {
		return model.UpdateMerchantPatch{}, "business_profile", errx.New(errx.CodeValidationFailed, "请完善业务身份并选择地图展示设置")
	}
	location, err := buildMerchantLocationPatch(req)
	if err != nil {
		return model.UpdateMerchantPatch{}, "location", err
	}
	profile, profileErr := model.NormalizeBusinessProfile(*req.BusinessProfile)
	if profileErr != nil {
		return model.UpdateMerchantPatch{}, "business_profile", errx.New(errx.CodeValidationFailed, profileErr.Error())
	}
	return model.UpdateMerchantPatch{
		Name: name, MainCategories: normalizeMerchantCategories(req.MainCategories),
		Description: strings.TrimSpace(req.Description), LogoURL: strings.TrimSpace(req.LogoURL), Images: append([]string(nil), req.Images...),
		ContactName: strings.TrimSpace(req.ContactName), ContactPhone: phone, ContactWechat: strings.TrimSpace(req.ContactWechat),
		PublicPhone: req.PublicPhone, PublicWechat: req.PublicWechat,
		BusinessProfile: &profile, MapVisible: req.MapVisible,
		AddressText: strings.TrimSpace(req.AddressText), Location: location,
	}, "", nil
}

// 主营品类只去空白和重复项，并保留用户填写顺序。
func normalizeMerchantCategories(values []string) []string {
	var result []string
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}
	return result
}

func buildMerchantLocationPatch(req UpdateMerchantReq) (model.MerchantLocationPatch, error) {
	action := strings.TrimSpace(req.LocationAction)
	switch action {
	case model.MerchantLocationActionKeep:
		return model.MerchantLocationPatch{Action: action}, nil
	case model.MerchantLocationActionClear:
		// clear 是完整清除操作，即使请求夹带旧坐标或档口文字也必须忽略，防止位置隐私残留。
		return model.MerchantLocationPatch{Action: action, Status: model.MerchantLocationStatusHidden}, nil
	case model.MerchantLocationActionSet:
		// 腾讯地图允许赤道或本初子午线上的合法 0 坐标，因此必须用指针区分缺失字段和显式 0。
		if req.Latitude == nil || req.Longitude == nil || !model.ValidMerchantCoordinates(*req.Latitude, *req.Longitude) {
			return model.MerchantLocationPatch{}, errx.New(errx.CodeValidationFailed, "地图位置格式不正确，请重新选择")
		}
		return model.MerchantLocationPatch{
			Action:       action,
			MarketName:   strings.TrimSpace(req.MarketName),
			BuildingName: strings.TrimSpace(req.BuildingName),
			FloorNo:      strings.TrimSpace(req.FloorNo),
			BoothNo:      strings.TrimSpace(req.BoothNo),
			Latitude:     *req.Latitude,
			Longitude:    *req.Longitude,
			LocationName: strings.TrimSpace(req.LocationName),
			Status:       model.MerchantLocationStatusVisible,
		}, nil
	default:
		return model.MerchantLocationPatch{}, errx.New(errx.CodeValidationFailed, "地图位置操作不正确，请重新选择")
	}
}

func isValidContactPhone(phone string) bool {
	if len(phone) < 6 || len(phone) > 20 {
		return false
	}
	for _, char := range phone {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

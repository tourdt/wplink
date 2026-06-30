package merchant

import (
	"strings"

	"wplink/backend/common/errx"
)

const misleadingMerchantNameMessage = "商家名称不能包含认证、官方等容易误导的字样"

var misleadingMerchantNameKeywords = []string{
	"认证",
	"官方",
	"平台推荐",
	"旗舰",
	"直营",
	"授权",
	"指定",
}

func validateMerchantName(name string) error {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil
	}
	// 商家名称由用户填写，不允许包含容易被理解为平台背书或官方授权的词，避免列表页误导买家。
	for _, keyword := range misleadingMerchantNameKeywords {
		if strings.Contains(trimmedName, keyword) {
			return errx.New(errx.CodeValidationFailed, misleadingMerchantNameMessage)
		}
	}
	return nil
}

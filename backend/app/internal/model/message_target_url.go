package model

import "net/url"

func MerchantMyResourcesTargetURL(merchantID string) string {
	values := url.Values{}
	values.Set("merchantId", merchantID)
	return "/pages/my-resources/index?" + values.Encode()
}

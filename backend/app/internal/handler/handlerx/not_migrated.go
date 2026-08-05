package handlerx

import (
	"net/http"

	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
)

// NotMigrated 只用于尚未接入生产 Server 的生成骨架，并以安全错误阻止接口被误认为可用。
func NotMigrated(handlerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只记录 Handler 标识，不记录 URL、请求头或请求体，避免敏感请求信息进入日志。
		logx.WithContext(r.Context()).Errorw("API Handler 尚未迁移", logx.Field("handler", handlerName))
		response.JSON(w, nil, errx.New(errx.CodeInternalError, "接口暂不可用，请稍后重试"))
	}
}

package {{.PkgName}}

import (
	"net/http"

	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"
)

{{if .HasDoc}}{{.Doc}}{{end}}
func {{.HandlerName}}(svcCtx *svc.ServiceContext) http.HandlerFunc {
	// 下列常量是代码生成门禁识别的稳定标记。
	// 骨架可编译且失败关闭，但提交前必须实现正式业务逻辑并删除此标记。
	const WPLINK_API_HANDLER_STUB = "{{.HandlerName}}"
	return func(w http.ResponseWriter, _ *http.Request) {
		_ = svcCtx
		_ = WPLINK_API_HANDLER_STUB
		response.JSON(w, nil, errx.New(errx.CodeInternalError, "接口暂不可用，请稍后重试"))
	}
}

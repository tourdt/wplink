package server

import (
	"net/http"
	"strings"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"
)

func registerAuthRoutes(mux *http.ServeMux, store authlogic.UserStore, tokenService authlogic.TokenService) {
	mux.HandleFunc("POST /api/v1/auth/wechat-login", func(w http.ResponseWriter, r *http.Request) {
		var body authlogic.WechatLoginReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := authlogic.NewWechatLoginLogic(store, tokenService).WechatLogin(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := authlogic.NewMeLogic(store).GetMe(r.Context(), userID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/me/phone", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body authlogic.BindPhoneReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := authlogic.NewMeLogic(store).BindPhone(r.Context(), userID, body)
		response.JSON(w, resp, err)
	})
}

func userIDFromBearerToken(r *http.Request, tokenService authlogic.TokenService) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", errx.New(errx.CodeUnauthorized, "请先登录")
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || token == header {
		return "", errx.New(errx.CodeUnauthorized, "请先登录")
	}
	subject, err := tokenService.ParseUserToken(r.Context(), token)
	if err != nil {
		return "", errx.New(errx.CodeUnauthorized, err.Error())
	}
	return subject.UserID, nil
}

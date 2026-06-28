package server

import (
	"net/http"
	"strings"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"
)

func registerAuthRoutes(mux *http.ServeMux, store authlogic.UserStore, tokenService authlogic.TokenService, wechatClient authlogic.WechatSessionClient, smsVerifier authlogic.SMSVerifier) {
	mux.HandleFunc("POST /api/v1/auth/wechat-login", func(w http.ResponseWriter, r *http.Request) {
		var body authlogic.WechatLoginReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := authlogic.NewWechatLoginLogic(store, tokenService, wechatClient).WechatLogin(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/auth/sms-code", func(w http.ResponseWriter, r *http.Request) {
		var body authlogic.SendSMSCodeReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		sender, _ := any(smsVerifier).(authlogic.SMSCodeSender)
		resp, err := authlogic.NewSendSMSCodeLogic(sender).SendSMSCode(r.Context(), body)
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
		resp, err := authlogic.NewMeLogic(store, smsVerifier).BindPhone(r.Context(), userID, body)
		response.JSON(w, resp, err)
	})
}

func userIDFromBearerToken(r *http.Request, tokenService authlogic.TokenService) (string, error) {
	subject, err := userSubjectFromBearerToken(r, tokenService)
	if err != nil {
		return "", err
	}
	return subject.UserID, nil
}

func optionalUserIDFromBearerToken(r *http.Request, tokenService authlogic.TokenService) (string, error) {
	if tokenService == nil {
		return "", nil
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", nil
	}
	// 公开行为埋点允许匿名记录；一旦请求携带 token，就必须使用服务端解析出的用户身份，避免前端伪造归因。
	subject, err := userSubjectFromBearerToken(r, tokenService)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(subject.UserID), nil
}

func userSubjectFromBearerToken(r *http.Request, tokenService authlogic.TokenService) (session.UserTokenSubject, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return session.UserTokenSubject{}, errx.New(errx.CodeUnauthorized, "请先登录")
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || token == header {
		return session.UserTokenSubject{}, errx.New(errx.CodeUnauthorized, "请先登录")
	}
	subject, err := tokenService.ParseUserToken(r.Context(), token)
	if err != nil {
		return session.UserTokenSubject{}, errx.New(errx.CodeUnauthorized, err.Error())
	}
	return subject, nil
}

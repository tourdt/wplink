package response

import (
	"encoding/json"
	"net/http"

	"wplink/backend/common/errx"
)

type envelope struct {
	Code      int         `json:"code"`
	ErrorCode string      `json:"errorCode,omitempty"`
	Msg       string      `json:"msg"`
	Data      interface{} `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, data interface{}, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err != nil {
		status := errx.HTTPStatus(err)
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(envelope{
			Code:      status,
			ErrorCode: errx.CodeOf(err),
			Msg:       errx.PublicMessage(err),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(envelope{
		Code: http.StatusOK,
		Msg:  "ok",
		Data: data,
	})
}

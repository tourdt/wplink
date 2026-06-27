package errx

import (
	"errors"
	"net/http"
)

const (
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeResourceNotFound = "RESOURCE_NOT_FOUND"
	CodeMerchantNotFound = "MERCHANT_NOT_FOUND"
	CodeStateConflict    = "STATE_CONFLICT"
	CodeQuotaNotEnough   = "QUOTA_NOT_ENOUGH"
	CodeReviewRequired   = "REVIEW_REQUIRED"
	CodeRateLimited      = "RATE_LIMITED"
	CodeInternalError    = "INTERNAL_ERROR"
)

const defaultPublicMessage = "操作失败，请稍后重试"

type Error struct {
	Code    string
	Message string
}

func New(code string, message string) error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func CodeOf(err error) string {
	var appErr *Error
	if errors.As(err, &appErr) && appErr.Code != "" {
		return appErr.Code
	}
	return CodeInternalError
}

func PublicMessage(err error) string {
	var appErr *Error
	if errors.As(err, &appErr) && appErr.Message != "" {
		return appErr.Message
	}
	return defaultPublicMessage
}

func HTTPStatus(err error) int {
	switch CodeOf(err) {
	case CodeValidationFailed:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeResourceNotFound, CodeMerchantNotFound:
		return http.StatusNotFound
	case CodeStateConflict, CodeQuotaNotEnough:
		return http.StatusConflict
	case CodeReviewRequired:
		return http.StatusUnprocessableEntity
	case CodeRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

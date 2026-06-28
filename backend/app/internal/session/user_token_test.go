package session

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestHMACUserTokenServiceParsesIssuedToken(t *testing.T) {
	service := NewHMACUserTokenService("secret", time.Hour)

	token, err := service.IssueUserToken(context.Background(), UserTokenSubject{UserID: "user-1", Roles: []string{"normal_user"}})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	subject, err := service.ParseUserToken(context.Background(), token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if subject.UserID != "user-1" || len(subject.Roles) != 1 || subject.Roles[0] != "normal_user" {
		t.Fatalf("subject = %#v, want user-1 normal_user", subject)
	}
}

func TestHMACUserTokenServiceRejectsTamperedToken(t *testing.T) {
	service := NewHMACUserTokenService("secret", time.Hour)
	token, err := service.IssueUserToken(context.Background(), UserTokenSubject{UserID: "user-1"})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	_, err = service.ParseUserToken(context.Background(), token+"x")
	if err == nil || !strings.Contains(err.Error(), "登录状态无效") {
		t.Fatalf("err = %v, want invalid login state", err)
	}
}

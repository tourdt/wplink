package session

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestHMACAdminTokenIssuerIssuesSignedToken(t *testing.T) {
	issuer := NewHMACAdminTokenIssuer("secret", time.Hour)

	token, err := issuer.IssueAdminToken(context.Background(), AdminTokenSubject{
		UserID: "user-1",
		Roles:  []string{"platform_operator"},
	})
	if err != nil {
		t.Fatalf("IssueAdminToken() error = %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d, want 3", len(parts))
	}
}

func TestHMACAdminTokenIssuerParsesSignedToken(t *testing.T) {
	issuer := NewHMACAdminTokenIssuer("secret", time.Hour)
	token, err := issuer.IssueAdminToken(context.Background(), AdminTokenSubject{
		UserID: "user-1",
		Roles:  []string{"platform_operator"},
	})
	if err != nil {
		t.Fatalf("IssueAdminToken() error = %v", err)
	}

	subject, err := issuer.ParseAdminToken(context.Background(), token)
	if err != nil {
		t.Fatalf("ParseAdminToken() error = %v", err)
	}

	if subject.UserID != "user-1" || len(subject.Roles) != 1 || subject.Roles[0] != "platform_operator" {
		t.Fatalf("subject = %#v, want issued subject", subject)
	}
}

func TestHMACAdminTokenIssuerRejectsEmptySecret(t *testing.T) {
	issuer := NewHMACAdminTokenIssuer("", time.Hour)

	_, err := issuer.IssueAdminToken(context.Background(), AdminTokenSubject{UserID: "user-1"})
	if err == nil {
		t.Fatal("IssueAdminToken() error = nil, want error")
	}
}

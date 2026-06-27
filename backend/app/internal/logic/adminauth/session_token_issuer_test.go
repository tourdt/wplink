package adminauth

import (
	"context"
	"testing"

	"wplink/backend/app/internal/session"
)

func TestSessionTokenIssuerPassesCredentialSubject(t *testing.T) {
	subjectIssuer := &recordingSubjectIssuer{token: "signed-token"}
	issuer := NewSessionTokenIssuer(subjectIssuer)

	token, err := issuer.IssueAdminToken(context.Background(), AdminCredential{
		UserID: "user-1",
		Roles:  []string{RolePlatformOperator},
	})
	if err != nil {
		t.Fatalf("IssueAdminToken() error = %v", err)
	}
	if token != "signed-token" {
		t.Fatalf("token = %q, want signed-token", token)
	}
	if subjectIssuer.subject.UserID != "user-1" {
		t.Fatalf("subject user = %q, want user-1", subjectIssuer.subject.UserID)
	}
	if len(subjectIssuer.subject.Roles) != 1 || subjectIssuer.subject.Roles[0] != RolePlatformOperator {
		t.Fatalf("subject roles = %#v, want platform operator", subjectIssuer.subject.Roles)
	}
}

type recordingSubjectIssuer struct {
	token   string
	subject session.AdminTokenSubject
}

func (i *recordingSubjectIssuer) IssueAdminToken(_ context.Context, subject session.AdminTokenSubject) (string, error) {
	i.subject = subject
	return i.token, nil
}

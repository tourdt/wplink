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
		OperatorID: "operator-1",
		Roles:      []string{RolePlatformOperator},
		Modules:    []string{"resource_review", "verification_review"},
	})
	if err != nil {
		t.Fatalf("IssueAdminToken() error = %v", err)
	}
	if token != "signed-token" {
		t.Fatalf("token = %q, want signed-token", token)
	}
	if subjectIssuer.subject.OperatorID != "operator-1" {
		t.Fatalf("subject operator = %q, want operator-1", subjectIssuer.subject.OperatorID)
	}
	if len(subjectIssuer.subject.Roles) != 1 || subjectIssuer.subject.Roles[0] != RolePlatformOperator {
		t.Fatalf("subject roles = %#v, want platform operator", subjectIssuer.subject.Roles)
	}
	if len(subjectIssuer.subject.Modules) != 2 || subjectIssuer.subject.Modules[0] != "resource_review" {
		t.Fatalf("subject modules = %#v, want credential modules", subjectIssuer.subject.Modules)
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

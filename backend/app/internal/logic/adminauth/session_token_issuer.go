package adminauth

import (
	"context"

	"wplink/backend/app/internal/session"
)

type subjectTokenIssuer interface {
	IssueAdminToken(ctx context.Context, subject session.AdminTokenSubject) (string, error)
}

type SessionTokenIssuer struct {
	issuer subjectTokenIssuer
}

func NewSessionTokenIssuer(issuer subjectTokenIssuer) *SessionTokenIssuer {
	return &SessionTokenIssuer{issuer: issuer}
}

func (i *SessionTokenIssuer) IssueAdminToken(ctx context.Context, credential AdminCredential) (string, error) {
	return i.issuer.IssueAdminToken(ctx, session.AdminTokenSubject{
		OperatorID:  credential.OperatorID,
		AuthVersion: credential.AuthVersion,
		Roles:       append([]string(nil), credential.Roles...),
		Modules:     append([]string(nil), credential.Modules...),
	})
}

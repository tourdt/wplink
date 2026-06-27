package errx

import (
	"errors"
	"testing"
)

func TestPublicMessageFallsBackToFriendlyInternalError(t *testing.T) {
	err := errors.New("sql: no rows in result set")

	got := PublicMessage(err)

	if got != "操作失败，请稍后重试" {
		t.Fatalf("PublicMessage() = %q, want 操作失败，请稍后重试", got)
	}
}

func TestPublicMessageUsesBusinessErrorMessage(t *testing.T) {
	err := New(CodeForbidden, "您没有权限进行此操作")

	got := PublicMessage(err)

	if got != "您没有权限进行此操作" {
		t.Fatalf("PublicMessage() = %q, want 您没有权限进行此操作", got)
	}
}

func TestCodeOfReturnsBusinessErrorCode(t *testing.T) {
	err := New(CodeStateConflict, "状态已变化，请刷新后重试")

	got := CodeOf(err)

	if got != CodeStateConflict {
		t.Fatalf("CodeOf() = %q, want %q", got, CodeStateConflict)
	}
}

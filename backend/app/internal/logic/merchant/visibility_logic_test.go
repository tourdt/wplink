package merchant

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"wplink/backend/common/errx"
)

type fakeVisibilityStore struct {
	called     bool
	merchantID string
	visible    bool
	err        error
}

func (s *fakeVisibilityStore) SetMerchantVisibility(_ context.Context, id string, visible bool) error {
	s.called = true
	s.merchantID = id
	s.visible = visible
	return s.err
}

func TestSetMerchantVisibilityRequiresExplicitBoolean(t *testing.T) {
	store := &fakeVisibilityStore{}
	if _, err := SetMerchantVisibility(context.Background(), store, "101", nil); errx.CodeOf(err) != errx.CodeValidationFailed || store.called {
		t.Fatalf("err = %v, called = %v", err, store.called)
	}
	visible := false
	id, err := SetMerchantVisibility(context.Background(), store, " 101 ", &visible)
	if err != nil || id != "101" || store.merchantID != "101" || store.visible {
		t.Fatalf("id = %q, store = %#v, err = %v", id, store, err)
	}
}

func TestSetMerchantVisibilityMapsStoreErrors(t *testing.T) {
	visible := true
	for _, tc := range []struct {
		err  error
		code string
	}{{sql.ErrNoRows, errx.CodeResourceNotFound}, {errors.New("internal details"), errx.CodeInternalError}} {
		store := &fakeVisibilityStore{err: tc.err}
		_, err := SetMerchantVisibility(context.Background(), store, "101", &visible)
		if errx.CodeOf(err) != tc.code {
			t.Fatalf("err = %v, want code %s", err, tc.code)
		}
	}
}

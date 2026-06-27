package adminauth

import "testing"

func TestBcryptPasswordHasherHashesAndVerifiesPassword(t *testing.T) {
	hasher := BcryptPasswordHasher{}

	hash, err := hasher.Hash("secret123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "" || hash == "secret123" {
		t.Fatalf("Hash() = %q, want non-empty hash different from raw password", hash)
	}
	if !hasher.Verify(hash, "secret123") {
		t.Fatal("Verify() = false, want true for correct password")
	}
	if hasher.Verify(hash, "wrong-password") {
		t.Fatal("Verify() = true, want false for wrong password")
	}
}

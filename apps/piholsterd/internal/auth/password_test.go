package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	ok, err := VerifyPassword("correct-horse-battery-staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("expected VerifyPassword to return true for correct password")
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	hash, err := HashPassword("my-secret-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Fatal("expected VerifyPassword to return false for wrong password")
	}
}

func TestHashFormat(t *testing.T) {
	hash, err := HashPassword("test-password-123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	// Must start with the correct PHC identifier.
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash does not start with $argon2id$: %q", hash)
	}

	// Must contain version 19.
	if !strings.Contains(hash, "v=19") {
		t.Fatalf("hash does not contain v=19: %q", hash)
	}

	// Must encode the correct parameters.
	if !strings.Contains(hash, "m=65536,t=3,p=2") {
		t.Fatalf("hash does not contain expected params m=65536,t=3,p=2: %q", hash)
	}

	// Structural check: exactly 5 $ separators → 6 parts.
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Fatalf("expected 6 parts in PHC format, got %d: %q", len(parts), hash)
	}

	// Salt and key must be non-empty.
	if parts[4] == "" {
		t.Fatal("salt part is empty")
	}
	if parts[5] == "" {
		t.Fatal("key part is empty")
	}
}

func TestUniqueHashesPerCall(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Fatal("two hashes of the same password must not be identical (salts must differ)")
	}
}

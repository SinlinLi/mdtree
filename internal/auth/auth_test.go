package auth

import (
	"testing"
	"time"
)

func TestPasswordHashing(t *testing.T) {
	const secret = "correct horse battery staple"
	hash, err := HashPassword(secret)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !VerifyPassword(hash, secret) {
		t.Error("VerifyPassword rejected the correct password")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Error("VerifyPassword accepted a wrong password")
	}
	if VerifyPassword("", "anything") {
		t.Error("VerifyPassword accepted input against an empty hash")
	}
}

func TestGeneratePasswordIsRandom(t *testing.T) {
	a, err := GeneratePassword(18)
	if err != nil {
		t.Fatalf("GeneratePassword: %v", err)
	}
	b, err := GeneratePassword(18)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("GeneratePassword produced two identical values")
	}
	if len(a) < 20 {
		t.Errorf("generated password is too short: %d chars", len(a))
	}
}

func TestSessionLifecycle(t *testing.T) {
	store := NewSessionStore(time.Hour)
	sess, err := store.Create()
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !store.Validate(sess.Token) {
		t.Error("a fresh session should validate")
	}
	if store.Validate("bogus-token") {
		t.Error("a bogus token should not validate")
	}
	store.Delete(sess.Token)
	if store.Validate(sess.Token) {
		t.Error("a deleted session should not validate")
	}
}

func TestSessionExpiry(t *testing.T) {
	store := NewSessionStore(-time.Second) // sessions expire immediately
	sess, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	if store.Validate(sess.Token) {
		t.Error("an expired session should not validate")
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	const ip = "10.0.0.1"
	for i := 0; i < 3; i++ {
		if !rl.Allowed(ip) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
		rl.Fail(ip)
	}
	if rl.Allowed(ip) {
		t.Error("the 4th attempt should be blocked")
	}
	rl.Reset(ip)
	if !rl.Allowed(ip) {
		t.Error("attempts should be allowed again after Reset")
	}
}

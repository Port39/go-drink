package users

import (
	"bytes"
	"testing"
)

func TestPassword(t *testing.T) {
	pass := "password"
	hash := CalculatePasswordHash(pass)
	hash2 := CalculatePasswordHash(pass)
	if bytes.Equal(hash, hash2) {
		t.Fail()
	}
	if bytes.Equal(hash[:32], hash2[:32]) {
		t.Fail()
	}
	if !VerifyPasswordHash(hash, pass) {
		t.Fail()
	}
	if !VerifyPasswordHash(hash2, pass) {
		t.Fail()
	}
	if VerifyPasswordHash(hash, "definitely wrong") {
		t.Fail()
	}
	if VerifyPasswordHash(hash2, "definitely wrong") {
		t.Fail()
	}
}

package users

import (
	"testing"
)

func TestEntropy(t *testing.T) {
	candidates := map[string]float64{
		"":         0.,
		"AAAA":     0.,
		"12345678": 0.375,
	}
	for candidate, expectedEntropy := range candidates {
		actualEntropy := Entropy([]byte(candidate))
		if actualEntropy != expectedEntropy {
			t.Fatalf("Entropy mismatch for '%s', expected: %f, got: %f", candidate, expectedEntropy, actualEntropy)
		}
	}
}

func TestHIBP(t *testing.T) {
	shouldFail := "password"
	shouldSucceed := "ajkoergbujiawogin"
	if !CheckHIBP(shouldFail) {
		t.Fail()
	}
	if CheckHIBP(shouldSucceed) {
		t.Fail()
	}
}

package users

import (
	"testing"
)

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

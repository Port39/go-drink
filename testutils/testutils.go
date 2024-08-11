package testutils

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"testing"
)

func GetEmptyDb(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	FailOnError(err, t)
	return db
}

func FailOnError(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func ExpectFailure(cond bool, t *testing.T) {
	t.Helper()
	if cond {
		t.Fail()
	}
}

func ExpectSuccess(cond bool, t *testing.T) {
	t.Helper()
	ExpectFailure(!cond, t)
}

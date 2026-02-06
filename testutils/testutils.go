package testutils

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	"github.com/Port39/go-drink/domain_errors"
	_ "modernc.org/sqlite"
)

func GetEmptyDb(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	FailOnError(err, t)
	return db
}

func ExpectError(err error, t *testing.T) {
	t.Helper()
	ExpectSuccess(err != nil, t)
}

func ExpectValidationErrorWithMessage(err *domain_errors.ValidationProblemDetail, msg string, t *testing.T) {
	t.Helper()
	ExpectSuccess(err != nil, t)

	ExpectSuccess(slices.ContainsFunc(err.Messages, func(m domain_errors.ValidationMessage) bool {
		return m.Message == msg
	}), t)
}

func ExpectErrorWithMessage(err error, msg string, t *testing.T) {
	t.Helper()
	ExpectError(err, t)
	if err.Error() != msg {
		t.Fatalf("Expected error '%s', but got '%s' instead.", msg, err.Error())
	}
}

func FailOnError(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func ExpectValidationError(err *domain_errors.ValidationProblemDetail, t *testing.T) {
	t.Helper()
	ExpectSuccess(err != nil, t)
}

func FailOnValidationError(err *domain_errors.ValidationProblemDetail, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func ExpectFailure(cond bool, t *testing.T) {
	t.Helper()
	if cond {
		t.FailNow()
	}
}

func ExpectSuccess(cond bool, t *testing.T) {
	t.Helper()
	ExpectFailure(!cond, t)
}

func ExpectEqual(got any, want any, t *testing.T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func GetTestingContext(t *testing.T) (context.Context, context.CancelFunc) {
	deadline, ok := t.Deadline()
	if ok {
		return context.WithDeadline(context.Background(), deadline)
	}
	return context.WithCancel(context.Background())
}

package handlehttp

import (
	"context"
	"github.com/Port39/go-drink/session"
	"log"
	"net/http"
)

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key string

const contextKeyStatus key = "status"
const contextKeySession key = "session"
const contextKeySessionToken key = "sessionToken"

type ContextStruct struct {
	Status     int
	HasError   bool
	Session    *session.Session
	HasSession bool
}

func CtxToStruct(ctx context.Context) ContextStruct {
	sess, ok := ContextGetSession(ctx)
	hasSession := ok && sess != nil
	var sessionRef *session.Session
	if hasSession {
		sessionRef = sess
	} else {
		sessionRef = nil
	}
	status, hasStatus := ContextGetStatus(ctx)
	if !hasStatus {
		log.Println("Could not find status in context, using 500 as default.")
		status = http.StatusInternalServerError
	}
	hasError := status >= 400

	return ContextStruct{
		Session:    sessionRef,
		HasSession: hasSession,
		Status:     status,
		HasError:   hasError,
	}
}

func ContextWithStatus(ctx context.Context, status int) context.Context {
	return context.WithValue(ctx, contextKeyStatus, status)
}
func ContextWithSession(ctx context.Context, session session.Session) context.Context {
	return context.WithValue(ctx, contextKeySession, &session)
}

func ContextWithoutSession(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKeySession, nil)
}

func ContextWithSessionToken(ctx context.Context, sessionToken string) context.Context {
	return context.WithValue(ctx, contextKeySessionToken, sessionToken)
}

func contextGet[T interface{}](ctx context.Context, key key) (T, bool) {
	u, ok := ctx.Value(key).(T)
	return u, ok
}

func ContextGetStatus(ctx context.Context) (int, bool) {
	return contextGet[int](ctx, contextKeyStatus)
}

func ContextGetSession(ctx context.Context) (*session.Session, bool) {
	return contextGet[*session.Session](ctx, contextKeySession)
}

func ContextGetSessionToken(ctx context.Context) (string, bool) {
	return contextGet[string](ctx, contextKeySessionToken)
}

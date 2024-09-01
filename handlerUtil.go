package main

import (
	"context"
	"embed"
	"github.com/Port39/go-drink/domain_errors"
	"github.com/Port39/go-drink/handlehttp"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/users"
	"io/fs"
	"net/http"
	"strings"
)

const ContextKeySessionToken = "SESSION_TOKEN"

func enrichRequestContext(next handlehttp.RequestHandler) handlehttp.RequestHandler {
	return func(r *http.Request) (int, any) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			return next(r)
		}

		split := strings.Split(authHeader, " ")
		if len(split) != 2 || split[0] != "Bearer" {
			return next(r)
		}
		return next(r.Clone(context.WithValue(r.Context(), ContextKeySessionToken, split[1])))
	}
}

func verifyRole(role string, next handlehttp.RequestHandler) handlehttp.RequestHandler {
	return func(r *http.Request) (int, any) {
		sessionToken := r.Context().Value(ContextKeySessionToken)
		if sessionToken == nil {
			return http.StatusUnauthorized, domain_errors.Unauthorized
		}

		s, err := sessionStore.Get(sessionToken.(string))
		if err != nil || !session.IsValid(&s) {
			return http.StatusUnauthorized, domain_errors.Unauthorized
		}

		if !users.CheckRole(s.Role, role) {
			return http.StatusForbidden, domain_errors.Forbidden
		}

		return next(r)
	}
}

//go:embed html-frontend/templates/*.gohtml
var rawHtmlTemplates embed.FS

func getHtmlTemplates() fs.FS {
	htmlTemplates, err := fs.Sub(rawHtmlTemplates, "html-frontend/templates")
	if err != nil {
		panic("HTML Templates not found!")
	}
	return htmlTemplates
}

var htmlTemplates fs.FS = getHtmlTemplates()

func toJsonOrHtmlByAccept(htmlPath string) handlehttp.GetResponseMapper {
	return handlehttp.MatchByAcceptHeader(
		handlehttp.ByAccept[handlehttp.ResponseMapper]{
			Json: handlehttp.JsonMapper,
			Html: handlehttp.HtmlMapper(htmlTemplates, htmlPath),
		},
	)
}

func handleEnhanced(path string, handler handlehttp.RequestHandler, getMapper handlehttp.GetResponseMapper) {
	http.HandleFunc(path, handlehttp.MappingResultOf(
		enrichRequestContext(handler),
		handlehttp.AddCorsHeader(handlehttp.CorsConfig{
			AddCorsHeader: config.AddCorsHeader,
			CorsWhitelist: config.CorsWhitelist,
		}, getMapper)),
	)
}

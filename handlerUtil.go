package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/Port39/go-drink/domain_errors"
	"github.com/Port39/go-drink/handlehttp"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/users"
)

const ContextKeySessionToken = "SESSION_TOKEN"

func enrichRequestContext(next handlehttp.RequestHandler) handlehttp.RequestHandler {
	return func(r *http.Request) (int, any) {
		var token string = ""

		authCookie, _ := r.Cookie(tokenCookieName)

		if authCookie != nil {
			token = authCookie.Value
		}

		if token == "" {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				return next(r)
			}

			split := strings.Split(authHeader, " ")
			if len(split) != 2 || split[0] != "Bearer" {
				return next(r)
			}
			token = split[1]
		}

		return next(r.Clone(context.WithValue(r.Context(), ContextKeySessionToken, token)))
	}
}

func verifyRole(role string, next handlehttp.RequestHandler) handlehttp.RequestHandler {
	return func(r *http.Request) (int, any) {
		sessionToken := r.Context().Value(ContextKeySessionToken)
		if sessionToken == nil {
			return domain_errors.Unauthorized()
		}

		s, err := sessionStore.Get(sessionToken.(string))
		if err != nil || !session.IsValid(&s) {
			return domain_errors.Unauthorized()
		}

		if !users.CheckRole(s.Role, role) {
			return domain_errors.Forbidden()
		}

		return next(r)
	}
}

//go:embed html-frontend/**/*.gohtml
var rawHtmlTemplates embed.FS

func getHtmlTemplates() fs.FS {
	htmlTemplates, err := fs.Sub(rawHtmlTemplates, "html-frontend")
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

func toHtml(htmlPath string) handlehttp.GetResponseMapper {
	return handlehttp.AlwaysMapWith(handlehttp.HtmlMapper(htmlTemplates, htmlPath))
}

var tokenCookieName = "__Host-Token"

func writeSessionCookie(mapper handlehttp.ResponseMapper) handlehttp.ResponseMapper {
	return func(w http.ResponseWriter, status int, data any) {
		switch d := data.(type) {
		case loginResponse:
			token := d.Token
			w.Header().Set("Set-Cookie", tokenCookieName+"="+token+";Secure;Same-Site=Strict;HttpOnly;Path=/")
		default:
		}
		mapper(w, status, data)
	}
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

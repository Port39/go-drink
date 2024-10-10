package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/Port39/go-drink/domain_errors"
	"github.com/Port39/go-drink/handlehttp"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/users"
)

func enrichRequestContext(next handlehttp.RequestHandler) handlehttp.RequestHandler {
	return func(r *http.Request) (context.Context, any) {
		var token = ""

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

		ctx := r.Context()

		s, err := sessionStore.Get(token)
		if err == nil && session.IsValid(&s) {
			ctx = handlehttp.ContextWithSession(ctx, s)
			ctx = handlehttp.ContextWithSessionToken(ctx, token)
		}

		return next(r.Clone(ctx))
	}
}

func verifyRole(role string, next handlehttp.RequestHandler) handlehttp.RequestHandler {
	return func(r *http.Request) (context.Context, any) {
		s, hasSession := handlehttp.ContextGetSession(r.Context())
		if !hasSession {
			log.Println("Rejecting request due to lacking session.")
			return handlehttp.ContextWithStatus(r.Context(), http.StatusUnauthorized), domain_errors.ForStatus(http.StatusUnauthorized)
		}

		if !users.CheckRole(s.Role, role) {
			log.Println("Rejecting request due to lacking role.")
			return handlehttp.ContextWithStatus(r.Context(), http.StatusForbidden), domain_errors.ForStatus(http.StatusForbidden)
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

var htmlTemplates = getHtmlTemplates()

func toJsonOrHtmlByAccept(htmlPath string) handlehttp.GetResponseMapper {
	return func(r *http.Request) handlehttp.ResponseMapper {
		mappersByAccept := handlehttp.ByAccept[handlehttp.GetResponseMapper]{
			Json: handlehttp.AlwaysMapWith(handlehttp.JsonMapper),
			Html: toHtml(htmlPath),
		}
		responseMapper := handlehttp.MatchByAcceptHeader(mappersByAccept, mappersByAccept.Json)(r)
		return responseMapper(r)
	}
}

func toHtml(htmlPath string) handlehttp.GetResponseMapper {
	return func(r *http.Request) handlehttp.ResponseMapper {
		isUnpolyRequest := r.Header.Get("X-Up-Version") != ""
		return handlehttp.HtmlMapper(htmlTemplates, isUnpolyRequest, htmlPath)
	}
}

//go:embed html-frontend/static/*
var rawStaticFiles embed.FS

func getStaticFiles() fs.FS {
	staticFiles, err := fs.Sub(rawStaticFiles, "html-frontend/static")
	if err != nil {
		panic("Static files not found!")
	}
	return staticFiles
}

var staticFiles = getStaticFiles()

var tokenCookieName = "__Host-Token"

func writeSessionCookie(getMapper handlehttp.GetResponseMapper) handlehttp.GetResponseMapper {
	return func(r *http.Request) handlehttp.ResponseMapper {
		return func(w http.ResponseWriter, input handlehttp.MappingInput) {
			switch d := input.Data.(type) {
			case loginResponse:
				token := d.Token
				w.Header().Set("Set-Cookie", tokenCookieName+"="+token+";Secure;Same-Site=Strict;HttpOnly;Path=/")
			default:
			}
			mapper := getMapper(r)
			mapper(w, input)
		}
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

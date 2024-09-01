package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Port39/go-drink/domain_errors"
	"github.com/Port39/go-drink/handlehttp"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/users"
	"io"
	"log"
	"net/http"
	"strings"
)

func addCorsHeader(next handlehttp.GetResponseMapper) handlehttp.GetResponseMapper {
	return func(r *http.Request) *handlehttp.ResponseMapper {
		mapper := next(r)
		var newMapper handlehttp.ResponseMapper = func(w http.ResponseWriter, data any) {
			if config.AddCorsHeader {
				w.Header().Set("Access-Control-Allow-Origin", config.CorsWhitelist)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			(*mapper)(w, data)
		}
		return &newMapper
	}
}

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

func activateJsonResponse(w http.ResponseWriter) {
	w.Header().Set("content-type", "application/json")
}

func logAndCreateError(message string, err error) error {
	log.Println(message, err)
	return errors.New(message)
}

type Validatable interface {
	Validate() error
}

func readValidJsonBody[T Validatable](r *http.Request) (T, error) {
	var req T

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		return req, logAndCreateError("error reading request body", err)
	}
	defer r.Body.Close()

	err = json.Unmarshal(rawBody, &req)

	if err != nil {
		return req, logAndCreateError("error unmarshalling json request body", err)
	}

	err = req.Validate()
	return req, err
}

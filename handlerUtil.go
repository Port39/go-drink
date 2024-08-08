package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/users"
	"io"
	"log"
	"net/http"
	"strings"
)

func addCorsHeader(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.AddCorsHeader {
			w.Header().Set("Access-Control-Allow-Origin", config.CorsWhitelist)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		next(w, r)
	}
}

func enrichRequestContext(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next(w, r)
			return
		}

		split := strings.Split(authHeader, " ")
		if len(split) != 2 || split[0] != "Bearer" {
			next(w, r)
			return
		}
		next(w, r.Clone(context.WithValue(r.Context(), ContextKeySessionToken, split[1])))
	}
}

func verifyRole(role string, next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionToken := r.Context().Value(ContextKeySessionToken)
		if sessionToken == nil {
			respondUnauthorized(w)
			return
		}

		s, err := sessionStore.Get(sessionToken.(string))
		if err != nil || !session.IsValid(&s) {
			respondUnauthorized(w)
			return
		}

		if !users.CheckRole(s.Role, role) {
			respondForbidden(w)
			return
		}
		next(w, r)
	}
}

func respondWithJson(w http.ResponseWriter, response any) {
	resp, err := json.Marshal(response)

	if err != nil {
		logAndRespondWithInternalError(w, "Error while creating json response:", err)
		return
	}

	activateJsonResponse(w)
	_, err = w.Write(resp)
}

func logAndRespondWithInternalError(w http.ResponseWriter, errMessage string, err error) {
	log.Println(errMessage, err)
	w.WriteHeader(http.StatusInternalServerError)
}

func respondBadRequest(w http.ResponseWriter, errMessage string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(errMessage))
}

func respondUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
}

func respondForbidden(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
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

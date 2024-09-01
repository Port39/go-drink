package handlehttp

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

func logAndCreateError(message string, err error) error {
	log.Println(message, err)
	return errors.New(message)
}

type Validatable interface {
	Validate() error
}

func ReadValidJsonBody[T Validatable](r *http.Request) (T, error) {
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

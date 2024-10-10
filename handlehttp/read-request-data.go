package handlehttp

import (
	"encoding/json"
	"errors"
	contenttype "github.com/Port39/go-drink/handlehttp/content-type"
	"github.com/gorilla/schema"
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

var decoder = schema.NewDecoder()

func readValidFormBody[T Validatable](r *http.Request) (T, error) {
	var req T

	err := r.ParseForm()

	if err != nil {
		return req, logAndCreateError("error parsing form", err)
	}

	err = decoder.Decode(&req, r.Form)

	if err != nil {
		return req, logAndCreateError("error decoding form", err)
	}

	err = req.Validate()
	return req, err
}

func ReadValidBody[T Validatable](r *http.Request) (T, error) {
	var req T
	mediatype, err := contenttype.GetMediaType(r)

	if err != nil {
		return req, logAndCreateError("error ascertaining content type", err)
	}

	if Json.Equal(mediatype) {
		return readValidJsonBody[T](r)
	} else {
		return readValidFormBody[T](r)
	}
}

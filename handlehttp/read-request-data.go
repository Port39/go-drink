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

func readValidJsonBody[T any](r *http.Request, dest *T) error {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		return logAndCreateError("error reading request body", err)
	}
	defer r.Body.Close()

	err = json.Unmarshal(rawBody, dest)

	if err != nil {
		return logAndCreateError("error unmarshalling json request body", err)
	}
	return nil
}

var decoder = schema.NewDecoder()

func readValidFormBody[T any](r *http.Request, dest *T) error {
	err := r.ParseForm()

	if err != nil {
		return logAndCreateError("error parsing form", err)
	}

	err = decoder.Decode(dest, r.Form)

	if err != nil {
		return logAndCreateError("error decoding form", err)
	}

	return nil
}

func ReadValidBody[T any, PT interface {
	Validatable
	*T
}](req *http.Request) (PT, error) {
	var parsed = new(T)
	mediatype, err := contenttype.GetMediaType(req)

	if err != nil {
		return nil, logAndCreateError("error ascertaining content type", err)
	}

	if Json.Equal(mediatype) {
		err = readValidJsonBody(req, parsed)
	} else {
		err = readValidFormBody(req, parsed)
	}

	if err != nil {
		return nil, logAndCreateError("error ascertaining content type", err)
	}

	return parsed, nil
}

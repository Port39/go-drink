package handlehttp

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/Port39/go-drink/domain_errors"
	contenttype "github.com/Port39/go-drink/handlehttp/content-type"
	"github.com/gorilla/schema"
)

func logAndCreateError(message string, err error) *domain_errors.ValidationProblemDetail {
	log.Println(message, err)

	valErr := domain_errors.NewValidationProblemDetail(
		domain_errors.ValidationMessage{
			Message: err.Error(),
		},
	)

	return &valErr
}

type Parseable[T any] interface {
	ValidateAndParse() (T, *domain_errors.ValidationProblemDetail)
}

func readValidJsonBody[T any](r *http.Request, dest *T) *domain_errors.ValidationProblemDetail {
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

func readValidFormBody[T any](r *http.Request, dest *T) *domain_errors.ValidationProblemDetail {
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

func ReadValidBody[T Parseable[T]](req *http.Request) (*T, *domain_errors.ValidationProblemDetail) {
	var parsed = new(T)
	mediatype, err := contenttype.GetMediaType(req)

	if err != nil {
		return nil, logAndCreateError("error ascertaining content type", err)
	}

	var valErr *domain_errors.ValidationProblemDetail
	if Json.Equal(mediatype) {
		valErr = readValidJsonBody(req, parsed)
	} else {
		valErr = readValidFormBody(req, parsed)
	}

	if valErr != nil {
		return nil, valErr
	}

	var validated T
	validated, validationErr := (*parsed).ValidateAndParse()

	return &validated, validationErr
}

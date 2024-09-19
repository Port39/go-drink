package handlehttp

import (
	"log"
	"net/http"

	contenttype "github.com/Port39/go-drink/handlehttp/content-type"
)

type GetByRequestMatching[T any] func(r *http.Request) *T

type ByAccept[T any] struct {
	Html T
	Json T
}

var Html = contenttype.MediaType{
	Type:    "text",
	Subtype: "html",
}

var Json = contenttype.MediaType{
	Type:    "application",
	Subtype: "json",
}

var AllowedMediaTypes = []contenttype.MediaType{
	Html, Json,
}

// Match the most appropriate choice in elementsByAccept based on the request's accept header
func MatchByAcceptHeader[T any](elementsByAccept ByAccept[T], defaultElement T) func(r *http.Request) T {
	return func(r *http.Request) T {
		result, _, err := contenttype.GetAcceptableMediaTypeFromHeader(r.Header.Get("Accept"), AllowedMediaTypes)

		if err != nil {
			log.Println("Accept header", r.Header.Get("Accept"), "=>", result, err)
			return defaultElement
		}

		if Html.Equal(result) {
			return elementsByAccept.Html
		}

		if Json.Equal(result) {
			return elementsByAccept.Json
		}

		return defaultElement
	}
}

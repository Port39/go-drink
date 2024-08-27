package handlehttp

import (
	"net/http"
)

type RequestResponseHandler func(w http.ResponseWriter, r *http.Request)
type RequestHandler func(r *http.Request) (result any)
type ResponseMapper func(w http.ResponseWriter, data any)
type GetResponseMapper func(r *http.Request) *ResponseMapper

func AlwaysMapWith(mapper ResponseMapper) GetResponseMapper {
	return func(r *http.Request) *ResponseMapper { return &mapper }
}

// Get a RequestResponseHandler by mapping the result of the RequestHandler using the ResponseMapper
func MappingResultOf(handler RequestHandler, getMapper GetResponseMapper) RequestResponseHandler {
	if handler == nil || getMapper == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		mapper := getMapper(r)
		if mapper == nil || *mapper == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		result := (handler)(r)
		(*mapper)(w, result)
	}
}

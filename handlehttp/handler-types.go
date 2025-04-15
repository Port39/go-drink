package handlehttp

import (
	"context"
	"net/http"
)

type RequestResponseHandler func(w http.ResponseWriter, r *http.Request)
type RequestHandler func(r *http.Request) (updatedContext context.Context, result any)
type MappingInput struct {
	Data any
	Ctx  ContextStruct
}
type ResponseMapper func(w http.ResponseWriter, input MappingInput)
type GetResponseMapper func(r *http.Request) ResponseMapper

func AlwaysMapWith(mapper ResponseMapper) GetResponseMapper {
	return func(r *http.Request) ResponseMapper { return mapper }
}

// MappingResultOf
// Get a RequestResponseHandler by mapping the result of the RequestHandler using the ResponseMapper
func MappingResultOf(handler RequestHandler, getMapper GetResponseMapper) RequestResponseHandler {
	if handler == nil || getMapper == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		mapper := getMapper(r)
		ctx, result := (handler)(r)
		(mapper)(w, MappingInput{
			Data: result,
			Ctx:  CtxToStruct(ctx),
		})
	}
}

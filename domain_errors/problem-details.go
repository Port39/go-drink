package domain_errors

import "net/http"

// See https://www.rfc-editor.org/rfc/rfc7807
type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

const DefaultProblemType = "about:blank"

type MultiProblemDetail struct {
	ProblemDetail
	Problems []ProblemDetail `json:"problems"`
}

const ValidationProblemType = "/problem-types/validation"

type ValidationProblemDetail struct {
	ProblemDetail
	Field string `json:"field"`
}

func ForStatus(status int) (int, ProblemDetail) {
	return status, ProblemDetail{
		Type:   DefaultProblemType,
		Title:  http.StatusText(status),
		Status: int(status),
	}
}

func Unauthorized() (int, ProblemDetail)        { return ForStatus(http.StatusUnauthorized) }
func Forbidden() (int, ProblemDetail)           { return ForStatus(http.StatusForbidden) }
func InternalServerError() (int, ProblemDetail) { return ForStatus(http.StatusInternalServerError) }
func NotFound() (int, ProblemDetail)            { return ForStatus(http.StatusNotFound) }

func ForStatusAndDetail(status int, detail string) (int, ProblemDetail) {
	status, result := ForStatus(status)
	result.Detail = detail
	return status, result
}

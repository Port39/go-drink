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

func ForStatus(status int) ProblemDetail {
	return ProblemDetail{
		Type:   DefaultProblemType,
		Title:  http.StatusText(status),
		Status: int(status),
	}
}

var Unauthorized ProblemDetail = ForStatus(http.StatusUnauthorized)
var Forbidden ProblemDetail = ForStatus(http.StatusForbidden)
var InternalServerError ProblemDetail = ForStatus(http.StatusInternalServerError)
var NotFound ProblemDetail = ForStatus(http.StatusNotFound)

func ForStatusAndDetail(status int, detail string) ProblemDetail {
	result := ForStatus(status)
	result.Detail = detail
	return result
}

package domain_errors

import (
	"net/http"
)

// ProblemDetail
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
		Status: status,
	}
}

func ForStatusAndDetail(status int, detail string) ProblemDetail {
	result := ForStatus(status)
	result.Detail = detail
	return result
}

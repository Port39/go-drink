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

const ValidationProblemType = "/problem-types/validation"

type ValidationProblemDetail struct {
	ProblemDetail
	Messages []ValidationMessage
}

type ValidationMessage struct {
	Field    string `json:"field"`
	Message  string `json:"detail"`
	Instance string `json:"instance"`
	Severity string `json:"severity"`
}

const SeverityWarning = "warning"
const SeverityError = "error"

func NewValidationProblemDetail(messages ...ValidationMessage) ValidationProblemDetail {
	return ValidationProblemDetail{
		ProblemDetail: ProblemDetail{
			Type:   ValidationProblemType,
			Title:  "Validation failed",
			Status: http.StatusBadRequest,
		},
		Messages: messages,
	}
}

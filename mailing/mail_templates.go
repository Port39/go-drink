package mailing

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed templates/passwordReset.txt
var passwordResetTemplate string

type passwordResetTemplateData struct {
	Username string
	Token    string
}

func applyPasswordResetTemplate(data passwordResetTemplateData) (string, error) {
	templ, _ := template.New("passwordReset").Parse(passwordResetTemplate)
	writer := new(bytes.Buffer)
	err := templ.Execute(writer, data)
	if err != nil {
		return "", err
	}

	return writer.String(), nil
}

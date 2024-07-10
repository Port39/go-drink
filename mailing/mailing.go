package mailing

import (
	"github.com/wneessen/go-mail"
)

var user string
var pass string
var server string
var port int
var from string

func Configure(username, password, host string, mailport int, fromAddress string) {
	server = host
	user = username
	pass = password
	from = fromAddress
	port = mailport
}

func SendPasswordResetTokenMail(username, email, token string) error {
	data := passwordResetTemplateData{
		Username: username,
		Token:    token,
	}
	message := mail.NewMsg()
	if err := message.From(from); err != nil {
		return err
	}
	if err := message.To(email); err != nil {
		return err
	}
	message.Subject("Password reset token")
	msg, err := applyPasswordResetTemplate(data)
	if err != nil {
		return err
	}
	message.SetBodyString(mail.TypeTextPlain, msg)
	client, err := mail.NewClient(server, mail.WithPort(port), mail.WithSSLPort(true), mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithUsername(user), mail.WithPassword(pass))
	if err != nil {
		return err
	}
	err = client.DialAndSend(message)
	return err
}

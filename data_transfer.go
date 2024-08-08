package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/Port39/go-drink/users"
	"github.com/google/uuid"
	"regexp"
)

var (
	USERNAME_REGEX = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,64}$`)
	EMAIL_REGEX    = regexp.MustCompile(`^[^@ \t\r\n]+@[^@ \t\r\n]+\.[^@ \t\r\n]+$`)
)

type passwordRegistrationRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (p *passwordRegistrationRequest) Validate() error {
	if !USERNAME_REGEX.MatchString(p.Username) {
		return errors.New("invalid username")
	}
	if p.Email != "" && !EMAIL_REGEX.MatchString(p.Email) {
		return errors.New("invalid email")
	}
	return validatePassword(p.Password)
}

func (p *passwordLoginRequest) Validate() error {
	return nil
}

type passwordLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (p *noneLoginRequest) Validate() error {
	return nil
}

type noneLoginRequest struct {
	Username string `json:"username"`
}

func (p *nfcLoginRequest) Validate() error {
	return nil
}

type nfcLoginRequest struct {
	Token string `json:"token"`
}

type loginResponse struct {
	Token      string `json:"token"`
	ValidUntil int64  `json:"validUntil"`
}

type addItemRequest struct {
	Name    string `json:"name"`
	Price   int    `json:"price"`
	Image   string `json:"image"`
	Amount  int    `json:"amount"`
	Barcode string `json:"barcode"`
}

func (r *addItemRequest) Validate() error {
	if len(r.Name) > 64 {
		return errors.New("name to long")
	}
	data, err := base64.StdEncoding.DecodeString(r.Image)
	if err != nil {
		return err
	}
	if len(data) > 2097152 {
		return errors.New("image to large (max 2MiB allowed)")
	}
	if r.Amount < 0 {
		return errors.New("amount must not be negative")
	}
	return nil
}

type updateItemRequest struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Price   int    `json:"price"`
	Image   string `json:"image"`
	Amount  int    `json:"amount"`
	Barcode string `json:"barcode"`
}

func (r *updateItemRequest) Validate() error {
	id, err := uuid.Parse(r.Id)
	if err != nil {
		return err
	}
	r.Id = id.String()

	if len(r.Name) > 64 {
		return errors.New("name to long")
	}
	data, err := base64.StdEncoding.DecodeString(r.Image)
	if err != nil {
		return err
	}
	if len(data) > 2097152 {
		return errors.New("image to large (max 2MiB allowed)")
	}
	if r.Amount < 0 {
		return errors.New("amount must not be negative")
	}
	return nil
}

type buyItemRequest struct {
	ItemId string `json:"itemId"`
	Amount int    `json:"amount"`
}

func (r *buyItemRequest) Validate() error {
	id, err := uuid.Parse(r.ItemId)
	if err != nil {
		return err
	}
	r.ItemId = id.String()
	if r.Amount < 1 {
		return errors.New("amount must be at least one item")
	}
	return nil
}

type addAuthMethodRequest struct {
	Method string `json:"method"`
	Data   string `json:"data"`
}

func (r *addAuthMethodRequest) Validate() error {
	if r.Method == "none" {
		return nil
	}
	if r.Method == "nfc" {
		if r.Data == "" {
			return errors.New("missing nfc uid")
		}
		_, err := hex.DecodeString(r.Data)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("invalid method")
}

func (r *changeCreditRequest) Validate() error {
	return nil
}

type changeCreditRequest struct {
	Diff int `json:"diff"`
}

type requestPasswordResetRequest struct {
	Username string `json:"username"`
}

func (p *requestPasswordResetRequest) Validate() error {
	if !USERNAME_REGEX.MatchString(p.Username) {
		return errors.New("invalid username")
	}
	return nil
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (p *resetPasswordRequest) Validate() error {
	token, err := uuid.Parse(p.Token)
	if err != nil {
		return err
	}
	p.Token = token.String()
	return validatePassword(p.Password)
}

func validatePassword(password string) error {
	if users.Entropy([]byte(password)) < 0.4 {
		return errors.New("the password is not random enough")
	}
	if users.CheckHIBP(password) {
		return errors.New("this password has been breached before")
	}
	return nil
}

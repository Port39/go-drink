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
	UsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,64}$`)
	EmailRegex    = regexp.MustCompile(`^[^@ \t\r\n]+@[^@ \t\r\n]+\.[^@ \t\r\n]+$`)
)

type passwordRegistrationRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (p passwordRegistrationRequest) ValidateAndParse() (passwordRegistrationRequest, error) {
	if !UsernameRegex.MatchString(p.Username) {
		return p, errors.New("invalid username")
	}
	if p.Email != "" && !EmailRegex.MatchString(p.Email) {
		return p, errors.New("invalid email")
	}
	return p, validatePassword(p.Password)
}

func (p passwordLoginRequest) ValidateAndParse() (passwordLoginRequest, error) {
	return p, nil
}

type passwordLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (p noneLoginRequest) ValidateAndParse() (noneLoginRequest, error) {
	return p, nil
}

type noneLoginRequest struct {
	Username string `json:"username"`
}

func (p nfcLoginRequest) ValidateAndParse() (nfcLoginRequest, error) {
	return p, nil
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

func (r addItemRequest) ValidateAndParse() (addItemRequest, error) {
	if len(r.Name) > 64 {
		return r, errors.New("name to long")
	}
	data, err := base64.StdEncoding.DecodeString(r.Image)
	if err != nil {
		return r, err
	}
	if len(data) > 2097152 {
		return r, errors.New("image to large (max 2MiB allowed)")
	}
	if r.Amount < 0 {
		return r, errors.New("amount must not be negative")
	}
	return r, nil
}

type updateItemRequest struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Price   int    `json:"price"`
	Image   string `json:"image"`
	Amount  int    `json:"amount"`
	Barcode string `json:"barcode"`
}

func (r updateItemRequest) ValidateAndParse() (updateItemRequest, error) {
	id, err := uuid.Parse(r.Id)
	if err != nil {
		return r, err
	}
	r.Id = id.String()

	if len(r.Name) > 64 {
		return r, errors.New("name too long")
	}
	data, err := base64.StdEncoding.DecodeString(r.Image)
	if err != nil {
		return r, err
	}
	if len(data) > 2097152 {
		return r, errors.New("image to large (max 2MiB allowed)")
	}
	if r.Amount < 0 {
		return r, errors.New("amount must not be negative")
	}
	return r, nil
}

type buyItemRequest struct {
	ItemId string `json:"itemId"`
	Amount int    `json:"amount"`
}

func (r buyItemRequest) ValidateAndParse() (buyItemRequest, error) {
	id, err := uuid.Parse(r.ItemId)
	if err != nil {
		return r, err
	}
	r.ItemId = id.String()
	if r.Amount < 1 {
		return r, errors.New("amount must be at least one item")
	}
	return r, nil
}

type addAuthMethodRequest struct {
	Method string `json:"method"`
	Data   string `json:"data"`
}

func (r addAuthMethodRequest) ValidateAndParse() (addAuthMethodRequest, error) {
	if r.Method == "none" {
		return r, nil
	}
	if r.Method == "nfc" {
		if r.Data == "" {
			return r, errors.New("missing nfc uid")
		}
		_, err := hex.DecodeString(r.Data)
		if err != nil {
			return r, err
		}
		return r, nil
	}
	return r, errors.New("invalid method")
}

func (r changeCreditRequest) ValidateAndParse() (changeCreditRequest, error) {
	return r, nil
}

type changeCreditRequest struct {
	Diff int `json:"diff"`
}

type requestPasswordResetRequest struct {
	Username string `json:"username"`
}

func (p requestPasswordResetRequest) ValidateAndParse() (requestPasswordResetRequest, error) {
	if !UsernameRegex.MatchString(p.Username) {
		return p, errors.New("invalid username")
	}
	return p, nil
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (p resetPasswordRequest) ValidateAndParse() (resetPasswordRequest, error) {
	token, err := uuid.Parse(p.Token)
	if err != nil {
		return p, err
	}
	p.Token = token.String()
	return p, validatePassword(p.Password)
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

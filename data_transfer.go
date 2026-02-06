package main

import (
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strconv"

	"github.com/Port39/go-drink/domain_errors"
	"github.com/Port39/go-drink/users"
	"github.com/google/uuid"
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

func (r passwordRegistrationRequest) ValidateAndParse() (passwordRegistrationRequest, *domain_errors.ValidationProblemDetail) {

	errs := checkUsername(r.Username, "username")
	errs = append(errs, checkEmail(r.Email, "email")...)
	errs = append(errs, validatePassword(r.Password)...)

	return withValidationErr(r, errs)
}

type passwordLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r passwordLoginRequest) ValidateAndParse() (passwordLoginRequest, *domain_errors.ValidationProblemDetail) {
	errs := []domain_errors.ValidationMessage{}
	return withValidationErr(r, errs)
}

type noneLoginRequest struct {
	Username string `json:"username"`
}

func (r noneLoginRequest) ValidateAndParse() (noneLoginRequest, *domain_errors.ValidationProblemDetail) {
	errs := []domain_errors.ValidationMessage{}
	return withValidationErr(r, errs)
}

type nfcLoginRequest struct {
	Token string `json:"token"`
}

func (r nfcLoginRequest) ValidateAndParse() (nfcLoginRequest, *domain_errors.ValidationProblemDetail) {
	errs := []domain_errors.ValidationMessage{}
	return withValidationErr(r, errs)
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

func (r addItemRequest) ValidateAndParse() (addItemRequest, *domain_errors.ValidationProblemDetail) {
	errs := checkLength(r.Name, 64, "name")
	errs = append(errs, checkDecodeImage(r.Image, 2097152, "image")...)
	errs = append(errs, checkGte(r.Amount, 1, "amount")...)

	return withValidationErr(r, errs)
}

type updateItemRequest struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Price   int    `json:"price"`
	Image   string `json:"image"`
	Amount  int    `json:"amount"`
	Barcode string `json:"barcode"`
}

func (r updateItemRequest) ValidateAndParse() (updateItemRequest, *domain_errors.ValidationProblemDetail) {
	id, errs := parseID(r.Id, "id")
	if len(errs) == 0 {
		r.Id = id
	}

	errs = append(errs, checkLength(r.Name, 64, "name")...)
	errs = append(errs, checkDecodeImage(r.Image, 2097152, "image")...)
	errs = append(errs, checkGte(r.Amount, 0, "amount")...)

	return withValidationErr(r, errs)
}

type buyItemRequest struct {
	ItemId string `json:"itemId"`
	Amount int    `json:"amount"`
}

func (r buyItemRequest) ValidateAndParse() (buyItemRequest, *domain_errors.ValidationProblemDetail) {
	id, errs := parseID(r.ItemId, "itemId")
	if len(errs) == 0 {
		r.ItemId = id
	}

	errs = append(errs, checkGte(r.Amount, 1, "amount")...)

	return withValidationErr(r, errs)
}

type addAuthMethodRequest struct {
	Method string `json:"method"`
	Data   string `json:"data"`
}

func (r addAuthMethodRequest) ValidateAndParse() (addAuthMethodRequest, *domain_errors.ValidationProblemDetail) {
	errs := []domain_errors.ValidationMessage{}

	if r.Method == "nfc" {
		if r.Data == "" {
			errs = append(errs, domain_errors.ValidationMessage{Severity: "error", Field: "Data", Message: "missing nfc uid"})
		}
		_, err := hex.DecodeString(r.Data)
		if err != nil {
			errs = append(errs, domain_errors.ValidationMessage{Severity: "error", Field: "Data", Message: "can't decode nfc uid"})
		}
	} else if r.Method != "none" {
		errs = append(errs, domain_errors.ValidationMessage{Severity: "error", Field: "", Message: "invalid method"})
	}

	return withValidationErr(r, errs)
}

type changeCreditRequest struct {
	Diff int `json:"diff"`
}

func (r changeCreditRequest) ValidateAndParse() (changeCreditRequest, *domain_errors.ValidationProblemDetail) {
	errs := []domain_errors.ValidationMessage{}
	return withValidationErr(r, errs)
}

type requestPasswordResetRequest struct {
	Username string `json:"username"`
}

func (r requestPasswordResetRequest) ValidateAndParse() (requestPasswordResetRequest, *domain_errors.ValidationProblemDetail) {
	errs := checkUsername(r.Username, "username")
	return withValidationErr(r, errs)
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (r resetPasswordRequest) ValidateAndParse() (resetPasswordRequest, *domain_errors.ValidationProblemDetail) {
	errs := []domain_errors.ValidationMessage{}

	token, err := uuid.Parse(r.Token)

	if err == nil {
		r.Token = token.String()
	} else {
		errs = append(errs, domain_errors.ValidationMessage{
			Severity: "error",
			Message:  "Wrong Token Syntax",
			Field:    "Token",
		})
	}

	errs = append(errs, validatePassword(r.Password)...)

	return withValidationErr(r, errs)
}

// ---- PRIVATES ----

func withValidationErr[T any](r T, errs []domain_errors.ValidationMessage) (T, *domain_errors.ValidationProblemDetail) {
	if len(errs) > 0 {
		validationErr := domain_errors.NewValidationProblemDetail(
			errs...,
		)
		return r, &validationErr
	}

	return r, nil
}

func parseID(id string, parName string) (string, []domain_errors.ValidationMessage) {
	errs := []domain_errors.ValidationMessage{}

	parsedID, err := uuid.Parse(id)

	if err == nil {
		id = parsedID.String()
	} else {
		errs = append(errs, domain_errors.ValidationMessage{Severity: "error", Field: parName, Message: parName + " is not a valid UUID"})
	}

	return id, errs
}

func checkLength(value string, maxLen int, parName string) []domain_errors.ValidationMessage {
	errs := []domain_errors.ValidationMessage{}
	if len(value) > maxLen {
		errs = append(errs, domain_errors.ValidationMessage{Severity: "error", Field: parName, Message: parName + " too long"})
	}
	return errs
}

func checkGte(value int, min int, parName string) []domain_errors.ValidationMessage {
	errs := []domain_errors.ValidationMessage{}

	if value < min {
		errs = append(errs, domain_errors.ValidationMessage{Severity: "error", Field: parName, Message: parName + " must not be smaller than " + strconv.Itoa(min)})
	}

	return errs
}
func checkUsername(username string, parName string) []domain_errors.ValidationMessage {
	errs := []domain_errors.ValidationMessage{}

	if !UsernameRegex.MatchString(username) {
		errs = append(errs, domain_errors.ValidationMessage{Severity: "error", Field: parName, Message: parName + " invalid"})
	}

	return errs
}
func checkEmail(email string, parName string) []domain_errors.ValidationMessage {
	errs := []domain_errors.ValidationMessage{}

	if email != "" && !EmailRegex.MatchString(email) {
		errs = append(errs, domain_errors.ValidationMessage{
			Field:    parName,
			Message:  parName + " invalid",
			Severity: "error",
		})
	}

	return errs
}

func checkDecodeImage(image string, maxLen int, parName string) []domain_errors.ValidationMessage {
	errs := []domain_errors.ValidationMessage{}

	data, err := base64.StdEncoding.DecodeString(image)
	if err != nil {
		errs = append(errs, domain_errors.ValidationMessage{Severity: "error", Field: parName, Message: "Image can't be decoded. Are you using Base64?"})
	}

	if len(data) > maxLen {
		errs = append(errs, domain_errors.ValidationMessage{Severity: "error", Field: parName, Message: "Image too large (max 2MiB allowed)"})
	}

	return errs
}

func validatePassword(password string) []domain_errors.ValidationMessage {
	errs := []domain_errors.ValidationMessage{}
	msg := ""

	if users.Entropy([]byte(password)) < 0.4 {
		msg = "The password is not random enough"
	}
	if users.CheckHIBP(password) {
		msg = "this password has been breached before"
	}
	if "" == msg {
		return nil
	}

	errs = append(errs,
		domain_errors.ValidationMessage{
			Field:    "password",
			Message:  msg,
			Severity: "error",
		},
	)

	return errs
}

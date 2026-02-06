package main

import (
	"strings"
	"testing"

	"github.com/Port39/go-drink/testutils"
)

const securePassword = "No need to check this, it is already verified in the users package"

func TestPasswordRegistrationRequest_ValidateAndParse(t *testing.T) {
	req := passwordRegistrationRequest{
		Username: "invalid user",
		Email:    "invalid email",
		Password: securePassword,
	}
	req, err := req.ValidateAndParse()
	testutils.ExpectValidationErrorWithMessage(err, "username invalid", t)
	req.Username = "ValidUser"
	req, err = req.ValidateAndParse()
	testutils.ExpectValidationErrorWithMessage(err, "email invalid", t)
	req.Email = "valid@godrink.test"
	req, err = req.ValidateAndParse()
	testutils.FailOnValidationError(err, t)
}

func TestAddItemRequest_ValidateAndParse(t *testing.T) {
	req := addItemRequest{
		Name:    "loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooong",
		Price:   0,
		Image:   "not base64 encoded data",
		Amount:  -1,
		Barcode: "",
	}
	req, err := req.ValidateAndParse()
	testutils.ExpectValidationErrorWithMessage(err, "name too long", t)
	req.Name = "not too long"
	req, err = req.ValidateAndParse()
	testutils.ExpectValidationError(err, t)
	req.Image = "AAAA"
	req, err = req.ValidateAndParse()
	testutils.ExpectValidationErrorWithMessage(err, "amount must not be smaller than 1", t)
	req.Amount = 1
	req, err = req.ValidateAndParse()
	testutils.FailOnValidationError(err, t)
}

func validUpdateRequest() updateItemRequest {
	return updateItemRequest{
		Id:      "00000000-0000-0000-0000-000000000000",
		Name:    "a name",
		Price:   1,
		Image:   "AAAA",
		Amount:  1,
		Barcode: "",
	}
}

func TestUpdateItemRequest_ValidateAndParse(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := validUpdateRequest()
		req, err := req.ValidateAndParse()
		testutils.FailOnValidationError(err, t)
	})

	t.Run("uuid without dashes is valid and will be normalized", func(t *testing.T) {
		req := validUpdateRequest()
		req.Id = strings.ReplaceAll(req.Id, "-", "")
		req, err := req.ValidateAndParse()
		testutils.FailOnValidationError(err, t)
		testutils.ExpectEqual(req.Id, validUpdateRequest().Id, t)
	})

	t.Run("id must be uuid", func(t *testing.T) {
		req := validUpdateRequest()
		req.Id = "not uuid"
		req, err := req.ValidateAndParse()
		testutils.ExpectValidationError(err, t)
	})

	t.Run("name can be 64 chars long", func(t *testing.T) {
		req := validUpdateRequest()
		req.Name = strings.Repeat("0", 64)
		req, err := req.ValidateAndParse()
		testutils.FailOnValidationError(err, t)
	})

	t.Run("name can be empty", func(t *testing.T) {
		req := validUpdateRequest()
		req.Name = ""
		req, err := req.ValidateAndParse()
		testutils.FailOnValidationError(err, t)
	})

	t.Run("name can't exceed 64 chars", func(t *testing.T) {
		req := validUpdateRequest()
		req.Name = strings.Repeat("0", 65)
		req, err := req.ValidateAndParse()
		testutils.ExpectValidationErrorWithMessage(err, "name too long", t)
	})

	t.Run("image must be valid base64", func(t *testing.T) {
		req := validUpdateRequest()
		req.Image = "bla"
		req, err := req.ValidateAndParse()
		testutils.ExpectValidationError(err, t)
	})

	t.Run("amount can be zero", func(t *testing.T) {
		req := validUpdateRequest()
		req.Amount = 0
		req, err := req.ValidateAndParse()
		testutils.FailOnValidationError(err, t)
	})

	t.Run("amount can't be less than zero", func(t *testing.T) {
		req := validUpdateRequest()
		req.Amount = -1
		req, err := req.ValidateAndParse()
		testutils.ExpectValidationErrorWithMessage(err, "amount must not be smaller than 0", t)
	})
}

func TestBuyItemRequest_ValidateAndParse(t *testing.T) {
	req := buyItemRequest{
		ItemId: "invalid itemId",
		Amount: 0,
	}
	req, err := req.ValidateAndParse()
	testutils.ExpectValidationError(err, t)
	req.ItemId = "00000000000000000000000000000000"
	req, err = req.ValidateAndParse()
	testutils.ExpectValidationErrorWithMessage(err, "amount must not be smaller than 1", t)
	testutils.ExpectSuccess(req.ItemId == "00000000-0000-0000-0000-000000000000", t)
	req.Amount = 1
	req, err = req.ValidateAndParse()
	testutils.FailOnValidationError(err, t)
}

func TestAddAuthMethodRequest_ValidateAndParse(t *testing.T) {
	req := addAuthMethodRequest{
		Method: "none",
		Data:   "",
	}
	req, err := req.ValidateAndParse()
	testutils.FailOnValidationError(err, t)

	req.Method = "nfc"
	req, err = req.ValidateAndParse()
	testutils.ExpectValidationErrorWithMessage(err, "missing nfc uid", t)
	req.Data = "invalid hex"
	req, err = req.ValidateAndParse()
	testutils.ExpectValidationError(err, t)
	req.Data = "deadbeef"
	req, err = req.ValidateAndParse()
	testutils.FailOnValidationError(err, t)

	req.Method = "password"
	req, err = req.ValidateAndParse()
	testutils.ExpectValidationErrorWithMessage(err, "invalid method", t)
}

func TestRequestPasswordResetRequest_ValidateAndParse(t *testing.T) {
	req := requestPasswordResetRequest{Username: "username invalid"}
	req, err := req.ValidateAndParse()
	testutils.ExpectValidationErrorWithMessage(err, "username invalid", t)
	req.Username = "valid_user"
	req, err = req.ValidateAndParse()
	testutils.FailOnValidationError(err, t)
}

func TestResetPasswordRequest_ValidateAndParse(t *testing.T) {
	req := resetPasswordRequest{
		Token:    "invalid uuid",
		Password: securePassword,
	}
	req, err := req.ValidateAndParse()
	testutils.ExpectValidationError(err, t)
	req.Token = "00000000000000000000000000000000"
	req, err = req.ValidateAndParse()
	testutils.FailOnValidationError(err, t)
	testutils.ExpectSuccess(req.Token == "00000000-0000-0000-0000-000000000000", t)
}

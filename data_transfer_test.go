package main

import (
	"strings"
	"testing"

	"github.com/Port39/go-drink/testutils"
)

const securePassword = "No need to check this, it is already verified in the users package"

func TestPasswordRegistrationRequest_Validate(t *testing.T) {
	req := passwordRegistrationRequest{
		Username: "invalid user",
		Email:    "invalid email",
		Password: securePassword,
	}
	testutils.ExpectErrorWithMessage(req.Validate(), "invalid username", t)
	req.Username = "ValidUser"
	testutils.ExpectErrorWithMessage(req.Validate(), "invalid email", t)
	req.Email = "valid@godrink.test"
	testutils.FailOnError(req.Validate(), t)
}

func TestAddItemRequest_Validate(t *testing.T) {
	req := addItemRequest{
		Name:    "loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooong",
		Price:   0,
		Image:   "not base64 encoded data",
		Amount:  -1,
		Barcode: "",
	}
	testutils.ExpectErrorWithMessage(req.Validate(), "name to long", t)
	req.Name = "not too long"
	testutils.ExpectError(req.Validate(), t)
	req.Image = "AAAA"
	testutils.ExpectErrorWithMessage(req.Validate(), "amount must not be negative", t)
	req.Amount = 1
	testutils.FailOnError(req.Validate(), t)
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

func TestUpdateItemRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := validUpdateRequest()
		testutils.FailOnError(req.Validate(), t)
	})

	t.Run("uuid without dashes is valid and will be normalized", func(t *testing.T) {
		req := validUpdateRequest()
		req.Id = strings.ReplaceAll(req.Id, "-", "")
		testutils.FailOnError(req.Validate(), t)
		testutils.ExpectEqual(req.Id, validUpdateRequest().Id, t)
	})

	t.Run("id must be uuid", func(t *testing.T) {
		req := validUpdateRequest()
		req.Id = "not uuid"
		testutils.ExpectError(req.Validate(), t)
	})

	t.Run("name can be 64 chars long", func(t *testing.T) {
		req := validUpdateRequest()
		req.Name = strings.Repeat("0", 64)
		testutils.FailOnError(req.Validate(), t)
	})

	t.Run("name can be empty", func(t *testing.T) {
		req := validUpdateRequest()
		req.Name = ""
		testutils.FailOnError(req.Validate(), t)
	})

	t.Run("name can't exceed 64 chars", func(t *testing.T) {
		req := validUpdateRequest()
		req.Name = strings.Repeat("0", 65)
		testutils.ExpectErrorWithMessage(req.Validate(), "name too long", t)
	})

	t.Run("image must be valid base64", func(t *testing.T) {
		req := validUpdateRequest()
		req.Image = "bla"
		testutils.ExpectError(req.Validate(), t)
	})

	t.Run("amount can be zero", func(t *testing.T) {
		req := validUpdateRequest()
		req.Amount = 0
		testutils.FailOnError(req.Validate(), t)
	})

	t.Run("amount can't be less than zero", func(t *testing.T) {
		req := validUpdateRequest()
		req.Amount = -1
		testutils.ExpectErrorWithMessage(req.Validate(), "amount must not be negative", t)
	})
}

func TestBuyItemRequest_Validate(t *testing.T) {
	req := buyItemRequest{
		ItemId: "invalid uuid",
		Amount: 0,
	}
	testutils.ExpectError(req.Validate(), t)
	req.ItemId = "00000000000000000000000000000000"
	testutils.ExpectErrorWithMessage(req.Validate(), "amount must be at least one item", t)
	testutils.ExpectSuccess(req.ItemId == "00000000-0000-0000-0000-000000000000", t)
	req.Amount = 1
	testutils.FailOnError(req.Validate(), t)
}

func TestAddAuthMethodRequest_Validate(t *testing.T) {
	req := addAuthMethodRequest{
		Method: "none",
		Data:   "",
	}
	testutils.FailOnError(req.Validate(), t)

	req.Method = "nfc"
	testutils.ExpectErrorWithMessage(req.Validate(), "missing nfc uid", t)
	req.Data = "invalid hex"
	testutils.ExpectError(req.Validate(), t)
	req.Data = "deadbeef"
	testutils.FailOnError(req.Validate(), t)

	req.Method = "password"
	testutils.ExpectErrorWithMessage(req.Validate(), "invalid method", t)
}

func TestRequestPasswordResetRequest_Validate(t *testing.T) {
	req := requestPasswordResetRequest{Username: "invalid username"}
	testutils.ExpectErrorWithMessage(req.Validate(), "invalid username", t)
	req.Username = "valid_user"
	testutils.FailOnError(req.Validate(), t)
}

func TestResetPasswordRequest_Validate(t *testing.T) {
	req := resetPasswordRequest{
		Token:    "invalid uuid",
		Password: securePassword,
	}
	testutils.ExpectError(req.Validate(), t)
	req.Token = "00000000000000000000000000000000"
	testutils.FailOnError(req.Validate(), t)
	testutils.ExpectSuccess(req.Token == "00000000-0000-0000-0000-000000000000", t)
}

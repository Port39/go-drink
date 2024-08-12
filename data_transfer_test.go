package main

import (
	"github.com/Port39/go-drink/testutils"
	"testing"
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

func TestUpdateItemRequest_Validate(t *testing.T) {
	req := updateItemRequest{
		Id:      "invalid uuid",
		Name:    "loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooong",
		Price:   0,
		Image:   "not base64 encoded data",
		Amount:  -1,
		Barcode: "",
	}
	testutils.ExpectError(req.Validate(), t)
	req.Id = "00000000000000000000000000000000"
	testutils.ExpectErrorWithMessage(req.Validate(), "name to long", t)
	testutils.ExpectSuccess(req.Id == "00000000-0000-0000-0000-000000000000", t)
	req.Name = "not too long"
	testutils.ExpectError(req.Validate(), t)
	req.Image = "AAAA"
	testutils.ExpectErrorWithMessage(req.Validate(), "amount must not be negative", t)
	req.Amount = 1
	testutils.FailOnError(req.Validate(), t)
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

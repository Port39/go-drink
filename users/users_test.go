package users

import (
	"bytes"
	"context"
	"database/sql"
	"github.com/Port39/go-drink/testutils"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"
)

var testUser1 = User{
	Id:       "00000000-0000-0000-0000-000000000001",
	Username: "test1",
	Email:    "test1@godrink.test",
	Role:     "user",
	Credit:   9001,
}

var testUser1NFCAuth = AuthenticationData{
	User: testUser1.Id,
	Type: "nfc",
	Data: []byte{0xde, 0xad, 0xbe, 0xef},
}

var testUser1PasswordAuth = AuthenticationData{
	User: testUser1.Id,
	Type: "password",
	Data: CalculatePasswordHash("password"),
}

var testUser2 = User{
	Id:       "00000000-0000-0000-0000-000000000002",
	Username: "test2",
	Email:    "test2@godrink.test",
	Role:     "user",
	Credit:   9001,
}

// mimics addPasswordResetToken, but the actual token is outdated and therefore invalid
func insertOutdatedPasswordResetToken(ctx context.Context, user *User, db *sql.DB) (PasswordResetToken, error) {
	var token PasswordResetToken
	token.UserId = user.Id
	token.Token = uuid.New().String()
	token.ValidUntil = time.Now().Add(-24 * time.Hour).Unix()
	_, err := db.ExecContext(ctx, `INSERT INTO password_reset (user_id, token, valid_until) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET token = $2, valid_until = $3`,
		token.UserId, token.Token, token.ValidUntil)
	if err != nil {
		return PasswordResetToken{}, err
	}
	return token, nil
}

func TestAuthenticationData_Equals(t *testing.T) {
	testData1 := AuthenticationData{
		User: "test",
		Type: "none",
		Data: []byte{0x13, 0x37},
	}
	testData2 := testData1
	testutils.ExpectSuccess(testData1.Equals(&testData2), t)

	testData2.Data = []byte{0x13, 0x38}
	testutils.ExpectFailure(testData1.Equals(&testData2), t)

	testData2.Data = []byte{0x42}
	testutils.ExpectFailure(testData1.Equals(&testData2), t)

	testData2.Data = []byte{0x13, 0x37}
	testData2.Type = "nfc"
	testutils.ExpectFailure(testData1.Equals(&testData2), t)

	testData2.User = "Another"
	testData2.Type = "none"
	testutils.ExpectFailure(testData1.Equals(&testData2), t)

	testData2.User = "test"
	testutils.ExpectSuccess(testData1.Equals(&testData2), t)
}

func TestPassword(t *testing.T) {
	pass := "password"
	hash := CalculatePasswordHash(pass)
	hash2 := CalculatePasswordHash(pass)

	testutils.ExpectFailure(bytes.Equal(hash, hash2), t)
	testutils.ExpectFailure(bytes.Equal(hash[:32], hash2[:32]), t)
	testutils.ExpectSuccess(VerifyPasswordHash(hash, pass), t)
	testutils.ExpectSuccess(VerifyPasswordHash(hash2, pass), t)
	testutils.ExpectFailure(VerifyPasswordHash(hash, "definitely wrong"), t)
	testutils.ExpectFailure(VerifyPasswordHash(hash2, "definitely wrong"), t)
}

func TestCashUser(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyUsersTableExists(db), t)

	// In an empty database, no cash user should be available
	_, err := GetUserForId(ctx, CASH_USER_ID, db)
	if err == nil || err.Error() != "no such user" {
		t.Fatal("Cash user exists, even though it wasn't created!")
	}

	// Create a cash user and verify their existence
	testutils.FailOnError(VerifyCashUserExists(db), t)
	cashUser, err := GetUserForId(ctx, CASH_USER_ID, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(cashUser.IsCashUser(), t)

	// Repeated calls to this function should not result in an error
	testutils.FailOnError(VerifyCashUserExists(db), t)
}

func TestAddUser(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyUsersTableExists(db), t)

	// Verify that the user hasn't been added before
	_, err := GetUserForId(ctx, testUser1.Id, db)
	if err == nil || err.Error() != "no such user" {
		t.Fatal("User exists, even though it wasn't created!")
	}

	_, err = GetUserForUsername(ctx, testUser1.Username, db)
	if err == nil || err.Error() != "no such user" {
		t.Fatal("User exists, even though it wasn't created!")
	}

	// Add testUser1. This should succeed.
	testutils.FailOnError(AddUser(ctx, testUser1, db), t)

	// Retrieve testUser1 from db by id
	retrievedUser, err := GetUserForId(ctx, testUser1.Id, db)
	testutils.FailOnError(err, t)
	if retrievedUser != testUser1 {
		t.Fatal("User retrieved from db is unequal to the original user.")
	}

	// Retrieve testUser1 from db by name
	retrievedUser, err = GetUserForUsername(ctx, testUser1.Username, db)
	testutils.FailOnError(err, t)
	if retrievedUser != testUser1 {
		t.Fatal("User retrieved from db is unequal to the original user.")
	}

	// Add testUser1 to the db a second time. This should fail, as they already exist.
	err = AddUser(ctx, testUser1, db)
	testutils.ExpectError(err, t)

	// However, adding another user should succeed
	testutils.FailOnError(AddUser(ctx, testUser2, db), t)

	allUsers, err := GetAllUsers(ctx, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(len(allUsers) == 2, t)
	testutils.ExpectSuccess(allUsers[0] == testUser1, t)
	testutils.ExpectSuccess(allUsers[1] == testUser2, t)
}

func TestGetUserForNFCToken(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyUsersTableExists(db), t)
	testutils.FailOnError(VerifyAuthTableExists(db), t)
	testutils.FailOnError(AddUser(ctx, testUser1, db), t)

	// No authentication data was added, so this should fail
	_, err := GetUserForNFCToken(ctx, testUser1NFCAuth.Data, db)
	testutils.ExpectError(err, t)

	testutils.FailOnError(AddAuthentication(ctx, testUser1NFCAuth, db), t)

	retrievedUser, err := GetUserForNFCToken(ctx, testUser1NFCAuth.Data, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(retrievedUser == testUser1, t)
}

func TestGetUsernamesWithNoneAuth(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyUsersTableExists(db), t)
	testutils.FailOnError(VerifyAuthTableExists(db), t)
	testutils.FailOnError(AddUser(ctx, testUser1, db), t)

	usernames, err := GetUsernamesWithNoneAuth(ctx, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(len(usernames) == 0, t)

	testutils.FailOnError(AddAuthentication(ctx, AuthenticationData{
		User: testUser1.Id,
		Type: "none",
		Data: nil,
	}, db), t)

	usernames, err = GetUsernamesWithNoneAuth(ctx, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(len(usernames) == 1, t)
	testutils.ExpectSuccess(usernames[0] == testUser1.Username, t)
}

func TestAddAuthenticationWithTransaction(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyUsersTableExists(db), t)
	testutils.FailOnError(VerifyAuthTableExists(db), t)
	testutils.FailOnError(AddUser(ctx, testUser1, db), t)

	tx, err := db.Begin()
	testutils.FailOnError(err, t)
	testutils.FailOnError(AddAuthenticationWithTransaction(ctx, testUser1NFCAuth, tx), t)
	testutils.FailOnError(tx.Rollback(), t)

	// since the transaction was canceled, this should not return auth data
	_, err = GetUserForNFCToken(ctx, testUser1NFCAuth.Data, db)
	testutils.ExpectError(err, t)

	tx, err = db.Begin()
	testutils.FailOnError(err, t)
	testutils.FailOnError(AddAuthenticationWithTransaction(ctx, testUser1NFCAuth, tx), t)
	testutils.FailOnError(tx.Commit(), t)

	// now it should succeed
	retrievedUser, err := GetUserForNFCToken(ctx, testUser1NFCAuth.Data, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(retrievedUser == testUser1, t)
}

func TestGetAuthForUser(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyAuthTableExists(db), t)

	_, err := GetAuthForUser(ctx, testUser1NFCAuth.User, testUser1NFCAuth.Type, db)
	testutils.ExpectError(err, t)

	testutils.FailOnError(AddAuthentication(ctx, testUser1NFCAuth, db), t)

	retrievedData, err := GetAuthForUser(ctx, testUser1NFCAuth.User, testUser1NFCAuth.Type, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(retrievedData.Equals(&testUser1NFCAuth), t)

	_, err = GetAuthForUser(ctx, testUser1NFCAuth.User, "none", db)
	testutils.ExpectError(err, t)
	_, err = GetAuthForUser(ctx, testUser1NFCAuth.User, "password", db)
	testutils.ExpectError(err, t)
}

func TestUpdateUser(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyUsersTableExists(db), t)
	testutils.FailOnError(AddUser(ctx, testUser1, db), t)

	newUsername := "UpdatedUsername"
	testUser1.Username = newUsername
	testutils.FailOnError(UpdateUser(ctx, &testUser1, db), t)

	retrievedUser, err := GetUserForId(ctx, testUser1.Id, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(retrievedUser.Username == newUsername, t)

	newBalance := 65535
	testUser1.Credit = newBalance
	tx, err := db.Begin()
	testutils.FailOnError(err, t)
	testutils.FailOnError(UpdateUserWithTransaction(ctx, &testUser1, tx), t)
	testutils.FailOnError(tx.Rollback(), t)

	retrievedUser, err = GetUserForId(ctx, testUser1.Id, db)
	testutils.FailOnError(err, t)
	testutils.ExpectFailure(retrievedUser.Credit == newBalance, t)
}

func TestCheckRole(t *testing.T) {
	// Admins are allowed to do anything
	testutils.ExpectSuccess(CheckRole("admin", "admin"), t)
	testutils.ExpectSuccess(CheckRole("admin", "user"), t)
	testutils.ExpectSuccess(CheckRole("admin", "doesnotexist"), t)

	// Accessing stuff from your role is okay
	testutils.ExpectSuccess(CheckRole("user", "user"), t)
	testutils.ExpectSuccess(CheckRole("doesnotexist", "doesnotexist"), t)

	// If you are not an admin, anything except your role is forbidden
	testutils.ExpectFailure(CheckRole("user", "admin"), t)
	testutils.ExpectFailure(CheckRole("user", "doesnotexist"), t)
	testutils.ExpectFailure(CheckRole("doesnotexist", "admin"), t)
	testutils.ExpectFailure(CheckRole("doesnotexist", "user"), t)
}

func TestAddPasswordResetToken(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyPasswordResetTableExists(db), t)

	token, err := addPasswordResetToken(ctx, &testUser1, db)
	testutils.FailOnError(err, t)

	// the token should have a lifetime of 24 hours (minus 30 seconds tolerance)
	testutils.ExpectSuccess((86400 >= (token.ValidUntil-time.Now().Unix())) &&
		((token.ValidUntil-time.Now().Unix()) > 86370), t)
	testutils.ExpectSuccess(token.UserId == testUser1.Id, t)

	retrievedToken, err := getPasswordResetDataByToken(ctx, token.Token, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(retrievedToken == token, t)

	// adding a new token for a user must replace the old one
	anotherToken, err := addPasswordResetToken(ctx, &testUser1, db)
	testutils.FailOnError(err, t)

	_, err = getPasswordResetDataByToken(ctx, token.Token, db)
	testutils.ExpectError(err, t)

	// the new token should still work though
	retrievedToken, err = getPasswordResetDataByToken(ctx, anotherToken.Token, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(retrievedToken == anotherToken, t)
}

func TestSendPasswordResetMail(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyUsersTableExists(db), t)
	testutils.FailOnError(VerifyCashUserExists(db), t)
	testutils.FailOnError(VerifyPasswordResetTableExists(db), t)

	cashuser, err := GetUserForId(ctx, CASH_USER_ID, db)
	testutils.FailOnError(err, t)

	err = SendPasswordResetMail(cashuser.Username, db)
	// Sending password reset mails for the cash user should always fail silently.
	testutils.FailOnError(err, t)

	testutils.FailOnError(AddUser(ctx, testUser1, db), t)
	err = SendPasswordResetMail(testUser1.Username, db)
	// since mailing is not set up, this should fail
	testutils.ExpectSuccess(strings.Contains(err.Error(), "mail: no address"), t)
}

func TestResetPassword(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyUsersTableExists(db), t)
	testutils.FailOnError(VerifyAuthTableExists(db), t)
	testutils.FailOnError(VerifyPasswordResetTableExists(db), t)

	testutils.FailOnError(AddUser(ctx, testUser1, db), t)
	testutils.FailOnError(AddAuthentication(ctx, testUser1PasswordAuth, db), t)

	invalidToken, err := insertOutdatedPasswordResetToken(ctx, &testUser1, db)
	testutils.FailOnError(err, t)
	err = ResetPassword(ctx, invalidToken.Token, "shouldn't work anyways", db)
	testutils.ExpectError(err, t)
	testutils.ExpectSuccess(err.Error() == "token expired", t)

	token, err := addPasswordResetToken(ctx, &testUser1, db)
	testutils.FailOnError(err, t)

	err = ResetPassword(ctx, token.Token, "changed", db)
	testutils.FailOnError(err, t)

	retrievedAuthData, err := GetAuthForUser(ctx, testUser1.Id, "password", db)
	testutils.FailOnError(err, t)

	testutils.ExpectFailure(VerifyPasswordHash(retrievedAuthData.Data, "password"), t)
	testutils.ExpectSuccess(VerifyPasswordHash(retrievedAuthData.Data, "changed"), t)
}

func TestDeleteResetToken(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyPasswordResetTableExists(db), t)

	token, err := addPasswordResetToken(ctx, &testUser1, db)
	testutils.FailOnError(err, t)

	testutils.FailOnError(DeleteResetToken(ctx, token.Token, db), t)

	_, err = getPasswordResetDataByToken(ctx, token.Token, db)
	testutils.ExpectError(err, t)
}

func TestDeleteResetTokenWithTransaction(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyPasswordResetTableExists(db), t)

	token, err := addPasswordResetToken(ctx, &testUser1, db)
	testutils.FailOnError(err, t)

	// Delete it, but rollback transaction
	tx, err := db.BeginTx(ctx, nil)
	testutils.FailOnError(err, t)

	err = DeleteResetTokenWithTransaction(ctx, token.Token, tx)
	testutils.FailOnError(err, t)

	err = tx.Rollback()
	testutils.FailOnError(err, t)

	retrievedToken, err := getPasswordResetDataByToken(ctx, token.Token, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(retrievedToken == token, t)

	// Now delete it for real
	tx, err = db.BeginTx(ctx, nil)
	testutils.FailOnError(err, t)

	err = DeleteResetTokenWithTransaction(ctx, token.Token, tx)
	testutils.FailOnError(err, t)

	err = tx.Commit()
	testutils.FailOnError(err, t)

	_, err = getPasswordResetDataByToken(ctx, token.Token, db)
	testutils.ExpectError(err, t)
}

func TestCleanExpiredResetTokens(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()
	ctx, cancel := testutils.GetTestingContext(t)
	defer cancel()

	testutils.FailOnError(VerifyPasswordResetTableExists(db), t)

	invalidToken, err := insertOutdatedPasswordResetToken(ctx, &testUser1, db)
	testutils.FailOnError(err, t)
	validToken, err := addPasswordResetToken(ctx, &testUser2, db)
	testutils.FailOnError(err, t)

	testutils.FailOnError(CleanExpiredResetTokens(ctx, db), t)

	_, err = getPasswordResetDataByToken(ctx, invalidToken.Token, db)
	testutils.ExpectError(err, t)

	retrievedToken, err := getPasswordResetDataByToken(ctx, validToken.Token, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(retrievedToken == validToken, t)
}

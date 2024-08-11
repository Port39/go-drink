package users

import (
	"bytes"
	"context"
	"github.com/Port39/go-drink/testutils"
	"testing"
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

var testUser2 = User{
	Id:       "00000000-0000-0000-0000-000000000002",
	Username: "test2",
	Email:    "test2@godrink.test",
	Role:     "user",
	Credit:   9001,
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

	testutils.FailOnError(VerifyUsersTableExists(db), t)

	// In an empty database, no cash user should be available
	_, err := GetUserForId(context.Background(), CASH_USER_ID, db)
	if err == nil || err.Error() != "no such user" {
		t.Fatal("Cash user exists, even though it wasn't created!")
	}

	// Create a cash user and verify their existence
	testutils.FailOnError(VerifyCashUserExists(db), t)
	cashUser, err := GetUserForId(context.Background(), CASH_USER_ID, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(cashUser.IsCashUser(), t)

	// Repeated calls to this function should not result in an error
	testutils.FailOnError(VerifyCashUserExists(db), t)
}

func TestAddUser(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()

	testutils.FailOnError(VerifyUsersTableExists(db), t)

	// Verify that the user hasn't been added before
	_, err := GetUserForId(context.Background(), testUser1.Id, db)
	if err == nil || err.Error() != "no such user" {
		t.Fatal("User exists, even though it wasn't created!")
	}

	_, err = GetUserForUsername(context.Background(), testUser1.Username, db)
	if err == nil || err.Error() != "no such user" {
		t.Fatal("User exists, even though it wasn't created!")
	}

	// Add testUser1. This should succeed.
	testutils.FailOnError(AddUser(context.Background(), testUser1, db), t)

	// Retrieve testUser1 from db by id
	retrievedUser, err := GetUserForId(context.Background(), testUser1.Id, db)
	testutils.FailOnError(err, t)
	if retrievedUser != testUser1 {
		t.Fatal("User retrieved from db is unequal to the original user.")
	}

	// Retrieve testUser1 from db by name
	retrievedUser, err = GetUserForUsername(context.Background(), testUser1.Username, db)
	testutils.FailOnError(err, t)
	if retrievedUser != testUser1 {
		t.Fatal("User retrieved from db is unequal to the original user.")
	}

	// Add testUser1 to the db a second time. This should fail, as they already exist.
	err = AddUser(context.Background(), testUser1, db)
	testutils.ExpectError(err, t)

	// However, adding another user should succeed
	testutils.FailOnError(AddUser(context.Background(), testUser2, db), t)

	allUsers, err := GetAllUsers(context.Background(), db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(len(allUsers) == 2, t)
	testutils.ExpectSuccess(allUsers[0] == testUser1, t)
	testutils.ExpectSuccess(allUsers[1] == testUser2, t)
}

func TestGetUserForNFCToken(t *testing.T) {
	db := testutils.GetEmptyDb(t)
	defer func() { testutils.FailOnError(db.Close(), t) }()

	testutils.FailOnError(VerifyUsersTableExists(db), t)
	testutils.FailOnError(VerifyAuthTableExists(db), t)
	testutils.FailOnError(AddUser(context.Background(), testUser1, db), t)

	// No authentication data was added, so this should fail
	_, err := GetUserForNFCToken(context.Background(), testUser1NFCAuth.Data, db)
	testutils.ExpectError(err, t)

	testutils.FailOnError(AddAuthentication(context.Background(), testUser1NFCAuth, db), t)

	retrievedUser, err := GetUserForNFCToken(context.Background(), testUser1NFCAuth.Data, db)
	testutils.FailOnError(err, t)
	testutils.ExpectSuccess(retrievedUser == testUser1, t)
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

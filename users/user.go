package users

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"github.com/Port39/go-drink/mailing"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"log"
	"strings"
	"time"
)

const CASH_USER_ID = "00000000-0000-0000-0000-000000000000"

type User struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Credit   int    `json:"credit"`
}

func (u *User) IsCashUser() bool {
	return u.Id == CASH_USER_ID
}

type AuthenticationData struct {
	User string
	Type string
	Data []byte
}

func (one *AuthenticationData) Equals(another *AuthenticationData) bool {
	if one.User != another.User {
		return false
	}
	if one.Type != another.Type {
		return false
	}
	if len(one.Data) != len(another.Data) {
		return false
	}
	for i, d := range one.Data {
		if another.Data[i] != d {
			return false
		}
	}
	return true
}

type PasswordResetToken struct {
	UserId     string
	Token      string
	ValidUntil int64
}

func CalculatePasswordHash(pass string) []byte {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		// fallback to a random uuid
		salt = []byte(strings.ReplaceAll(uuid.New().String(), "-", ""))
	}
	hash := argon2.IDKey([]byte(pass), salt, 1, 64*1024, 4, 32)
	return append(salt, hash...)
}

func VerifyPasswordHash(hash []byte, pass string) bool {
	salt := hash[:32]
	targetKey := hash[32:]
	actualKey := argon2.IDKey([]byte(pass), salt, 1, 64*1024, 4, 32)
	return bytes.Equal(targetKey, actualKey)
}

func VerifyUsersTableExists(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
    		id VARCHAR (36) PRIMARY KEY,
    		username VARCHAR (64) UNIQUE NOT NULL,
    		email VARCHAR (64),
    		role VARCHAR (16),
    		credit INTEGER
		)`)
	return err
}

func VerifyCashUserExists(db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO users (id, username, email, role, credit) 
	VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, CASH_USER_ID, "CASH PAYMENTS", "cash@localhost", "user", 65535)
	return err
}

func VerifyAuthTableExists(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS auth (
    		user_id VARCHAR (36) NOT NULL,
    		type VARCHAR (16) NOT NULL,
    		data bytea,
    		PRIMARY KEY (user_id, type)
		)`)
	return err
}

func VerifyPasswordResetTableExists(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS password_reset (
    		user_id VARCHAR (36) UNIQUE NOT NULL,
    		token VARCHAR (36) PRIMARY KEY,
    		valid_until BIGINT
		)`)
	return err
}

func GetUserForId(ctx context.Context, id string, db *sql.DB) (User, error) {
	result, err := db.QueryContext(ctx, "SELECT id, username, email, role, credit FROM users WHERE id = $1", id)
	if err != nil {
		return User{}, err
	}
	defer result.Close()
	if !result.Next() {
		return User{}, errors.New("no such user")
	}
	var user User
	err = result.Scan(&user.Id, &user.Username, &user.Email, &user.Role, &user.Credit)
	return user, err
}

func GetUserForUsername(ctx context.Context, username string, db *sql.DB) (User, error) {
	result, err := db.QueryContext(ctx, "SELECT id, username, email, role, credit FROM users WHERE username = $1", username)
	if err != nil {
		return User{}, err
	}
	defer result.Close()
	if !result.Next() {
		return User{}, errors.New("no such user")
	}
	var user User
	err = result.Scan(&user.Id, &user.Username, &user.Email, &user.Role, &user.Credit)
	return user, err
}

func GetUserForNFCToken(ctx context.Context, token []byte, db *sql.DB) (User, error) {
	result, err := db.QueryContext(ctx, `SELECT user_id FROM auth WHERE type = 'nfc' AND data = $1`, token)
	if err != nil {
		return User{}, err
	}
	if !result.Next() {
		return User{}, errors.New("invalid token")
	}
	var userId string
	err = result.Scan(&userId)
	if err != nil {
		return User{}, err
	}
	err = result.Close()
	if err != nil {
		return User{}, err
	}
	return GetUserForId(ctx, userId, db)
}

func GetUsernamesWithNoneAuth(ctx context.Context, db *sql.DB) ([]string, error) {
	result, err := db.QueryContext(ctx, `SELECT user_id FROM auth WHERE type = 'none'`)
	if err != nil {
		return nil, err
	}
	userIds := make([]string, 0)
	for result.Next() {
		var userId string
		err = result.Scan(&userId)
		if err != nil {
			continue
		}
		userIds = append(userIds, userId)
	}
	err = result.Close()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for _, userId := range userIds {
		user, err := GetUserForId(ctx, userId, db)
		if err != nil {
			continue
		}
		names = append(names, user.Username)
	}
	return names, nil
}

func AddUser(ctx context.Context, user User, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "INSERT INTO users (id, username, email, role, credit) VALUES ($1, $2, $3, $4, $5)",
		user.Id, user.Username, user.Email, user.Role, user.Credit)
	return err
}

func AddAuthentication(ctx context.Context, auth AuthenticationData, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "INSERT INTO auth (user_id, type, data) VALUES ($1, $2, $3) ON CONFLICT(user_id, type) DO UPDATE SET data = $3",
		auth.User, auth.Type, auth.Data)
	return err
}

func AddAuthenticationWithTransaction(ctx context.Context, auth AuthenticationData, tr *sql.Tx) error {
	_, err := tr.ExecContext(ctx, "INSERT INTO auth (user_id, type, data) VALUES ($1, $2, $3) ON CONFLICT(user_id, type) DO UPDATE SET data = $3",
		auth.User, auth.Type, auth.Data)
	return err
}

func GetAuthForUser(ctx context.Context, id, authtype string, db *sql.DB) (AuthenticationData, error) {
	result, err := db.QueryContext(ctx, "SELECT user_id, type, data FROM auth WHERE user_id = $1 AND type = $2", id, authtype)
	if err != nil {
		return AuthenticationData{}, err
	}
	defer result.Close()
	if !result.Next() {
		return AuthenticationData{}, errors.New("no matching authentication available")
	}
	var auth AuthenticationData
	err = result.Scan(&auth.User, &auth.Type, &auth.Data)
	return auth, err
}

func GetAllUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	users := make([]User, 0)
	result, err := db.QueryContext(ctx, "SELECT id, username, email, role, credit FROM users")
	if err != nil {
		return users, err
	}
	defer result.Close()
	for result.Next() {
		var user User
		err = result.Scan(&user.Id, &user.Username, &user.Email, &user.Role, &user.Credit)
		if err != nil {
			log.Println("Error reading results:", err)
		}
		users = append(users, user)
	}
	return users, nil
}

func UpdateUserWithTransaction(ctx context.Context, user *User, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `UPDATE users SET username = $1, email = $2, role = $3, credit = $4 WHERE id = $5`,
		user.Username, user.Email, user.Role, user.Credit, user.Id)
	return err
}

func UpdateUser(ctx context.Context, user *User, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `UPDATE users SET username = $1, email = $2, role = $3, credit = $4 WHERE id = $5`,
		user.Username, user.Email, user.Role, user.Credit, user.Id)
	return err
}

func CheckRole(actual, target string) bool {
	if actual == "admin" || actual == target {
		return true
	}
	return false
}

func addPasswordResetToken(ctx context.Context, user *User, db *sql.DB) (PasswordResetToken, error) {
	var token PasswordResetToken
	token.UserId = user.Id
	token.Token = uuid.New().String()
	token.ValidUntil = time.Now().Add(24 * time.Hour).Unix()
	_, err := db.ExecContext(ctx, `INSERT INTO password_reset (user_id, token, valid_until) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET token = $2, valid_until = $3`,
		token.UserId, token.Token, token.ValidUntil)
	if err != nil {
		return PasswordResetToken{}, err
	}
	return token, nil
}

func SendPasswordResetMail(username string, db *sql.DB) error {
	ctx := context.Background()
	user, err := GetUserForUsername(ctx, username, db)
	if err != nil {
		return err
	}
	if user.Id == CASH_USER_ID {
		return nil
	}
	token, err := addPasswordResetToken(ctx, &user, db)
	if err != nil {
		return err
	}
	err = mailing.SendPasswordResetTokenMail(user.Username, user.Email, token.Token)
	return err
}

func ResetPassword(ctx context.Context, token string, password string, db *sql.DB) error {
	tokenData, err := getPasswordResetDataByToken(ctx, token, db)
	if err != nil {
		return err
	}
	if time.Now().Unix() > tokenData.ValidUntil {
		_ = DeleteResetToken(ctx, token, db)
		return errors.New("token expired")
	}
	user, err := GetUserForId(ctx, tokenData.UserId, db)
	if err != nil {
		return err
	}
	auth := AuthenticationData{
		User: user.Id,
		Type: "password",
		Data: CalculatePasswordHash(password),
	}
	tr, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	err = DeleteResetTokenWithTransaction(ctx, token, tr)
	if err != nil {
		if tr.Rollback() != nil {
			return err
		}
		return err
	}
	err = AddAuthenticationWithTransaction(ctx, auth, tr)
	if err != nil {
		if tr.Rollback() != nil {
			return err
		}
		return err
	}
	return tr.Commit()
}

func getPasswordResetDataByToken(ctx context.Context, token string, db *sql.DB) (PasswordResetToken, error) {
	result, err := db.QueryContext(ctx, `SELECT user_id, token, valid_until FROM password_reset WHERE token = $1`, token)
	defer result.Close()
	if err != nil {
		return PasswordResetToken{}, err
	}
	if !result.Next() {
		return PasswordResetToken{}, errors.New("unknown token, it might have expired")
	}
	var resetToken PasswordResetToken
	err = result.Scan(&resetToken.UserId, &resetToken.Token, &resetToken.ValidUntil)
	return resetToken, err
}

func DeleteResetToken(ctx context.Context, token string, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DELETE FROM password_reset WHERE token = $1`, token)
	return err
}

func DeleteResetTokenWithTransaction(ctx context.Context, token string, tr *sql.Tx) error {
	_, err := tr.ExecContext(ctx, `DELETE FROM password_reset WHERE token = $1`, token)
	return err
}

func CleanExpiredResetTokens(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DELETE FROM password_reset WHERE valid_until <= $1`, time.Now().Unix())
	return err
}

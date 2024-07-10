package users

import (
	"bytes"
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

func GetUserForId(id string, db *sql.DB) (User, error) {
	result, err := db.Query("SELECT id, username, email, role, credit FROM users WHERE id = $1", id)
	defer result.Close()
	if err != nil {
		return User{}, err
	}
	if !result.Next() {
		return User{}, errors.New("no such user")
	}
	var user User
	err = result.Scan(&user.Id, &user.Username, &user.Email, &user.Role, &user.Credit)
	return user, err
}

func GetUserForUsername(username string, db *sql.DB) (User, error) {
	result, err := db.Query("SELECT id, username, email, role, credit FROM users WHERE username = $1", username)
	defer result.Close()
	if err != nil {
		return User{}, err
	}
	if !result.Next() {
		return User{}, errors.New("no such user")
	}
	var user User
	err = result.Scan(&user.Id, &user.Username, &user.Email, &user.Role, &user.Credit)
	return user, err
}

func GetUserForNFCToken(token []byte, db *sql.DB) (User, error) {
	result, err := db.Query(`SELECT user_id FROM auth WHERE type = 'nfc' AND data = $1`, token)
	defer result.Close()
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
	return GetUserForId(userId, db)
}

func GetUsernamesWithNoneAuth(db *sql.DB) ([]string, error) {
	result, err := db.Query(`SELECT user_id FROM auth WHERE type = 'none'`)
	defer result.Close()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for result.Next() {
		var userId string
		err = result.Scan(&userId)
		if err != nil {
			continue
		}
		user, err := GetUserForId(userId, db)
		if err != nil {
			continue
		}
		names = append(names, user.Username)
	}
	return names, nil
}

func AddUser(user User, db *sql.DB) error {
	_, err := db.Exec("INSERT INTO users (id, username, email, role, credit) VALUES ($1, $2, $3, $4, $5)",
		user.Id, user.Username, user.Email, user.Role, user.Credit)
	return err
}

func AddAuthentication(auth AuthenticationData, db *sql.DB) error {
	_, err := db.Exec("INSERT INTO auth (user_id, type, data) VALUES ($1, $2, $3) ON CONFLICT(user_id, type) DO UPDATE SET data = $3",
		auth.User, auth.Type, auth.Data)
	return err
}

func AddAuthenticationWithTransaction(auth AuthenticationData, tr *sql.Tx) error {
	_, err := tr.Exec("INSERT INTO auth (user_id, type, data) VALUES ($1, $2, $3) ON CONFLICT(user_id, type) DO UPDATE SET data = $3",
		auth.User, auth.Type, auth.Data)
	return err
}

func GetAuthForUser(id, authtype string, db *sql.DB) (AuthenticationData, error) {
	result, err := db.Query("SELECT user_id, type, data FROM auth WHERE user_id = $1 AND type = $2", id, authtype)
	defer result.Close()
	if err != nil {
		return AuthenticationData{}, err
	}
	if !result.Next() {
		return AuthenticationData{}, errors.New("no matching authentication available")
	}
	var auth AuthenticationData
	err = result.Scan(&auth.User, &auth.Type, &auth.Data)
	return auth, err
}

func GetAllUsers(db *sql.DB) ([]User, error) {
	users := make([]User, 0)
	result, err := db.Query("SELECT id, username, email, role, credit FROM users")
	defer result.Close()
	if err != nil {
		return users, err
	}
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

func UpdateUserWithTransaction(user *User, tx *sql.Tx) error {
	_, err := tx.Exec(`UPDATE users SET username = $1, email = $2, role = $3, credit = $4 WHERE id = $5`,
		user.Username, user.Email, user.Role, user.Credit, user.Id)
	return err
}

func UpdateUser(user *User, db *sql.DB) error {
	_, err := db.Exec(`UPDATE users SET username = $1, email = $2, role = $3, credit = $4 WHERE id = $5`,
		user.Username, user.Email, user.Role, user.Credit, user.Id)
	return err
}

func CheckRole(actual, target string) bool {
	if actual == "admin" || actual == target {
		return true
	}
	return false
}

func addPasswordResetToken(user *User, db *sql.DB) (PasswordResetToken, error) {
	var token PasswordResetToken
	token.UserId = user.Id
	token.Token = uuid.New().String()
	token.ValidUntil = time.Now().Add(24 * time.Hour).Unix()
	_, err := db.Exec(`INSERT INTO password_reset (user_id, token, valid_until) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET token = $2, valid_until = $3`,
		token.UserId, token.Token, token.ValidUntil)
	if err != nil {
		return PasswordResetToken{}, err
	}
	return token, nil
}

func SendPasswordResetMail(username string, db *sql.DB) error {
	user, err := GetUserForUsername(username, db)
	if err != nil {
		return err
	}
	if user.Id == CASH_USER_ID {
		return nil
	}
	token, err := addPasswordResetToken(&user, db)
	if err != nil {
		return err
	}
	err = mailing.SendPasswordResetTokenMail(user.Username, user.Email, token.Token)
	return err
}

func ResetPassword(token string, password string, db *sql.DB) error {
	tokenData, err := getPasswordResetDataByToken(token, db)
	if err != nil {
		return err
	}
	if time.Now().Unix() > tokenData.ValidUntil {
		_ = DeleteResetToken(token, db)
		return errors.New("token expired")
	}
	user, err := GetUserForId(tokenData.UserId, db)
	if err != nil {
		return err
	}
	auth := AuthenticationData{
		User: user.Id,
		Type: "password",
		Data: CalculatePasswordHash(password),
	}
	tr, err := db.Begin()
	if err != nil {
		return err
	}
	err = DeleteResetTokenWithTransaction(token, tr)
	if err != nil {
		if tr.Rollback() != nil {
			return err
		}
		return err
	}
	err = AddAuthenticationWithTransaction(auth, tr)
	if err != nil {
		if tr.Rollback() != nil {
			return err
		}
		return err
	}
	return tr.Commit()
}

func getPasswordResetDataByToken(token string, db *sql.DB) (PasswordResetToken, error) {
	result, err := db.Query(`SELECT user_id, token, valid_until FROM password_reset WHERE token = $1`, token)
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

func DeleteResetToken(token string, db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM password_reset WHERE token = $1`, token)
	return err
}

func DeleteResetTokenWithTransaction(token string, tr *sql.Tx) error {
	_, err := tr.Exec(`DELETE FROM password_reset WHERE token = $1`, token)
	return err
}

func CleanExpiredResetTokens(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM password_reset WHERE valid_until <= $1`, time.Now().Unix())
	return err
}

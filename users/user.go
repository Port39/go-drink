package users

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"log"
	"strings"
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
	_, err := db.Exec("INSERT INTO auth (user_id, type, data) VALUES ($1, $2, $3) ON CONFLICT(user_id, type) DO UPDATE SET data = $4",
		auth.User, auth.Type, auth.Data, auth.Data)
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

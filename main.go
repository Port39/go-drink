package main

import (
	"database/sql"
	"fmt"
	"github.com/Port39/go-drink/items"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/transactions"
	"github.com/Port39/go-drink/users"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DbConnectionString string
	Port               int
	SessionLifetime    int
}

var config Config
var database *sql.DB
var sessionStore session.Store

func mkconf() Config {
	portString, exists := os.LookupEnv("GODRINK_PORT")
	port := 8080
	var err error
	if exists {
		port, err = strconv.Atoi(portString)
		if err != nil {
			port = 8080
			log.Println(fmt.Sprintf("Error parsing port from env, defaulting to %d:", port), err)
		}
	}

	dbstring, exists := os.LookupEnv("GODRINK_DB")
	if !exists {
		log.Fatal("No database given, exiting!")
	}
	lifetime := 300
	lifetimeString, exists := os.LookupEnv("GODRINK_SESSIONLIFETIME")
	if exists {
		lifetime, err = strconv.Atoi(lifetimeString)
		if err != nil {
			lifetime = 300
			log.Println(fmt.Sprintf("Error parsing session lifetime from env, defaulting to %d:", lifetime), err)
		}
	}
	return Config{
		DbConnectionString: dbstring,
		Port:               port,
		SessionLifetime:    lifetime,
	}
}

func initialize() {
	config = mkconf()
	db, err := sql.Open("postgres", config.DbConnectionString)
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}
	database = db
	err = items.VerifyItemsTableExists(database)
	if err != nil {
		log.Fatal("Error creating items table: ", err)
	}
	err = users.VerifyUsersTableExists(database)
	if err != nil {
		log.Fatal("Error creating users table: ", err)
	}
	err = users.VerifyCashUserExists(database)
	if err != nil {
		log.Fatal("Error creating cash payments user:", err)
	}
	err = users.VerifyAuthTableExists(database)
	if err != nil {
		log.Fatal("Error creating auth table: ", err)
	}
	err = transactions.VerifyTransactionTableExists(database)
	if err != nil {
		log.Fatal("Error creating transaction table: ", err)
	}
	sessionStore = session.NewMemoryStore()
	cleanupTicker := time.NewTicker(time.Duration(config.SessionLifetime) * time.Second)
	go func() {
		for {
			select {
			case t := <-cleanupTicker.C:
				log.Println("Triggering session purge at:", t)
				sessionStore.Purge()
			}
		}
	}()
}

func main() {

	initialize()

	http.HandleFunc("GET /items", getItems)
	http.HandleFunc("GET /items/{id}", getItem)
	http.HandleFunc("POST /items/add", verifyRole("admin", addItem))
	http.HandleFunc("POST /items/update", verifyRole("admin", updateItem))

	http.HandleFunc("GET /users", verifyRole("admin", getUsers))
	http.HandleFunc("GET /users/noauth", getUsersWithNoneAuth)
	http.HandleFunc("GET /users/{id}", verifyRole("admin", getUser))

	http.HandleFunc("POST /register/password", registerWithPassword)

	http.HandleFunc("POST /auth/add", verifyRole("user", addAuthMethod))

	http.HandleFunc("POST /login/password", loginWithPassword)
	http.HandleFunc("POST /login/cash", loginCash)
	http.HandleFunc("POST /login/none", loginNone)
	http.HandleFunc("POST /login/nfc", loginNFC)

	http.HandleFunc("POST /logout", logout)

	http.HandleFunc("POST /buy", verifyRole("user", buyItem))

	http.HandleFunc("GET /transactions", verifyRole("admin", getTransactions))

	http.HandleFunc("POST /credit", verifyRole("user", changeCredit))

	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", config.Port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

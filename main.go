package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Port39/go-drink/items"
	"github.com/Port39/go-drink/mailing"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/transactions"
	"github.com/Port39/go-drink/users"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const ContextKeySessionToken = "SESSION_TOKEN"

type Config struct {
	DbConnectionString string
	Port               int
	SessionLifetime    int
	MailHost           string
	MailPort           int
	MailLogin          string
	MailPassword       string
	MailFrom           string
	AddCorsHeader      bool
	CorsWhitelist      string
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
	smtpserver, exists := os.LookupEnv("GODRINK_SMTPHOST")
	var mailHost string
	mailPort := 465
	if !exists {
		log.Println("No SMTP server given, mailing will fail!")
	} else {
		split := strings.Split(smtpserver, ":")
		if len(split) != 2 {
			log.Println("SMTP server must be specified as <host:port>, expect errors!")
		} else {
			mailHost = split[0]
			mailPort, err = strconv.Atoi(split[1])
			if err != nil {
				log.Println("SMTP server must be specified as <host:port>, expect errors!")
			}
		}
	}
	mailLogin, exists := os.LookupEnv("GODRINK_SMTPUSER")
	if !exists {
		log.Println("No SMTP username given, mailing will likely fail!")
	}
	mailPass, exists := os.LookupEnv("GODRINK_SMTPPASS")
	if !exists {
		log.Println("No SMTP password given, mailing will likely fail!")
	}
	mailFrom, exists := os.LookupEnv("GODRINK_SMTPFROM")
	if !exists {
		mailFrom = mailLogin
	}
	cors, addCorsHeader := os.LookupEnv("GODRINK_CORS")

	return Config{
		DbConnectionString: dbstring,
		Port:               port,
		SessionLifetime:    lifetime,
		MailHost:           mailHost,
		MailPort:           mailPort,
		MailLogin:          mailLogin,
		MailPassword:       mailPass,
		MailFrom:           mailFrom,
		AddCorsHeader:      addCorsHeader,
		CorsWhitelist:      cors,
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
	err = users.VerifyPasswordResetTableExists(database)
	if err != nil {
		log.Fatal("Error creating password reset token table: ", err)
	}
	err = transactions.VerifyTransactionTableExists(database)
	if err != nil {
		log.Fatal("Error creating transaction table: ", err)
	}
	databaseCleanupTicker := time.NewTicker(4 * time.Hour)
	go func() {
		for {
			select {
			case t := <-databaseCleanupTicker.C:
				log.Println("Starting Database Cleanup at:", t.Format(time.DateTime))
				if err := users.CleanExpiredResetTokens(context.Background(), database); err != nil {
					log.Println("Error while deleting expired password reset tokens:", err)
				}
			}
		}
	}()

	mailing.Configure(config.MailLogin, config.MailPassword, config.MailHost, config.MailPort, config.MailFrom)

	sessionStore = session.NewMemoryStore()
	sessionCleanupTicker := time.NewTicker(time.Duration(config.SessionLifetime) * time.Second)
	go func() {
		for {
			select {
			case t := <-sessionCleanupTicker.C:
				log.Println("Triggering session purge at:", t.Format(time.DateTime))
				sessionStore.Purge()
			}
		}
	}()
}

func main() {

	initialize()

	http.HandleFunc("GET /items", addCorsHeader(enrichRequestContext(getItems)))
	http.HandleFunc("GET /items/{id}", addCorsHeader(enrichRequestContext(getItem)))
	http.HandleFunc("POST /items/add", addCorsHeader(enrichRequestContext(verifyRole("admin", addItem))))
	http.HandleFunc("POST /items/update", addCorsHeader(enrichRequestContext(verifyRole("admin", updateItem))))
	http.HandleFunc("GET /items/barcode/{id}", addCorsHeader(enrichRequestContext(getItemByBarcode)))

	http.HandleFunc("GET /users", addCorsHeader(enrichRequestContext(verifyRole("admin", getUsers))))
	http.HandleFunc("GET /users/noauth", addCorsHeader(enrichRequestContext(getUsersWithNoneAuth)))
	http.HandleFunc("GET /users/{id}", addCorsHeader(enrichRequestContext(verifyRole("admin", getUser))))

	http.HandleFunc("POST /register/password", addCorsHeader(enrichRequestContext(registerWithPassword)))

	http.HandleFunc("POST /auth/add", addCorsHeader(enrichRequestContext(verifyRole("user", addAuthMethod))))
	http.HandleFunc("POST /auth/password-reset/request", addCorsHeader(enrichRequestContext(requestPasswordReset)))
	http.HandleFunc("POST /auth/password-reset", addCorsHeader(enrichRequestContext(resetPassword)))

	http.HandleFunc("POST /login/password", addCorsHeader(enrichRequestContext(loginWithPassword)))
	http.HandleFunc("POST /login/cash", addCorsHeader(enrichRequestContext(loginCash)))
	http.HandleFunc("POST /login/none", addCorsHeader(enrichRequestContext(loginNone)))
	http.HandleFunc("POST /login/nfc", addCorsHeader(enrichRequestContext(loginNFC)))

	http.HandleFunc("POST /logout", addCorsHeader(enrichRequestContext(logout)))

	http.HandleFunc("POST /buy", addCorsHeader(enrichRequestContext(verifyRole("user", buyItem))))

	http.HandleFunc("GET /transactions", addCorsHeader(enrichRequestContext(verifyRole("admin", getTransactions))))

	http.HandleFunc("POST /credit", addCorsHeader(enrichRequestContext(verifyRole("user", changeCredit))))

	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", config.Port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

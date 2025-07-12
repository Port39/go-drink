package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Port39/go-drink/handlehttp"
	"github.com/Port39/go-drink/items"
	"github.com/Port39/go-drink/mailing"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/transactions"
	"github.com/Port39/go-drink/users"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

var database *sql.DB
var sessionStore session.Store

func initialize() {
	config = mkconf()
	db, err := sql.Open(config.DbDriver, config.DbConnectionString)
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
	err = users.VerifyAdminUserExists(database)
	if err != nil {
		log.Fatal("Error creating admin user: ", err)
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

var noData handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), nil
}

func main() {
	initialize()

	http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFiles)))

	handleEnhanced("GET /index", noData, toHtml("templates/index.gohtml"))
	handleEnhanced("GET /", noData, toHtml("templates/index.gohtml"))
	handleEnhanced("GET /login", noData, toHtml("templates/login.gohtml"))

	handleEnhanced("GET /items", getItems, toJsonOrHtmlByAccept("templates/items.gohtml"))

	handleEnhanced("GET /items/{id}", getItem, handlehttp.AlwaysMapWith(handlehttp.JsonMapper))
	handleEnhanced("POST /items/add", verifyRole("admin", addItem), toJsonOrHtmlByAccept("templates/new-item.gohtml"))
	handleEnhanced("POST /items/update", verifyRole("admin", updateItem), handlehttp.AlwaysMapWith(handlehttp.JsonMapper))
	handleEnhanced("GET /items/barcode/{id}", getItemByBarcode, handlehttp.AlwaysMapWith(handlehttp.JsonMapper))

	handleEnhanced("GET /users", verifyRole("admin", getUsers), toJsonOrHtmlByAccept("templates/users.gohtml"))
	handleEnhanced("GET /users/noauth", getUsersWithNoneAuth, handlehttp.AlwaysMapWith(handlehttp.JsonMapper))
	handleEnhanced("GET /users/{id}", verifyRole("admin", getUser), handlehttp.AlwaysMapWith(handlehttp.JsonMapper))

	handleEnhanced("POST /register/password", registerWithPassword, writeSessionCookie(toJsonOrHtmlByAccept("templates/index.gohtml")))

	handleEnhanced("POST /auth/add", verifyRole("user", addAuthMethod), handlehttp.AlwaysMapWith(handlehttp.JsonMapper))
	handleEnhanced("POST /auth/password-reset/request", requestPasswordReset, handlehttp.AlwaysMapWith(handlehttp.JsonMapper))
	handleEnhanced("POST /auth/password-reset", resetPassword, handlehttp.AlwaysMapWith(handlehttp.JsonMapper))

	handleEnhanced("POST /login/password", loginWithPassword, writeSessionCookie(toJsonOrHtmlByAccept("templates/index.gohtml")))
	handleEnhanced("POST /login/cash", loginCash, handlehttp.AlwaysMapWith(handlehttp.JsonMapper))
	handleEnhanced("POST /login/none", loginNone, handlehttp.AlwaysMapWith(handlehttp.JsonMapper))
	handleEnhanced("POST /login/nfc", loginNFC, handlehttp.AlwaysMapWith(handlehttp.JsonMapper))

	handleEnhanced("POST /logout", logout, writeSessionCookie(toJsonOrHtmlByAccept("templates/index.gohtml")))

	handleEnhanced("POST /buy", verifyRole("user", buyItem), handlehttp.AlwaysMapWith(handlehttp.JsonMapper))

	handleEnhanced("GET /transactions", verifyRole("admin", getTransactions), handlehttp.AlwaysMapWith(handlehttp.JsonMapper))

	handleEnhanced("POST /credit", verifyRole("user", changeCredit), handlehttp.AlwaysMapWith(handlehttp.JsonMapper))

	uri := fmt.Sprintf("0.0.0.0:%d", config.Port)
	log.Println("Serving go-drink on " + uri)
	err := http.ListenAndServe(uri, nil)
	if err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// The SQLITE_DRIVER value comes from modernc.org/sqlite/sqlite.driverName
const SQLITE_DRIVER = "sqlite"

type Config struct {
	DbDriver           string
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

	dbdriver, driverExists := os.LookupEnv("GODRINK_DBDRIVER")
	dbUrl, dbUrlExists := os.LookupEnv("GODRINK_DB")
	if !driverExists {
		log.Println("No database driver given, using embedded sqlite.")
		log.Println("##############################################################")
		log.Println("# Caution: NO DATA WILL PERSIST ACROSS APPLICATION RESTARTS! #")
		log.Println("##############################################################")
		dbdriver = SQLITE_DRIVER
		dbUrl = "file::memory:?cache=shared"
	} else {
		dbdriver = strings.ToLower(dbdriver)
		if !dbUrlExists {
			if dbdriver == SQLITE_DRIVER {
				dbUrl = "file::memory:?cache=shared"
			} else {
				log.Fatalf("The database driver (%s) requires specifying a connection string!", dbdriver)
			}
		}
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
		DbDriver:           dbdriver,
		DbConnectionString: dbUrl,
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

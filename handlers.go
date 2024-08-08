package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Port39/go-drink/items"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/transactions"
	"github.com/Port39/go-drink/users"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func addCorsHeader(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.AddCorsHeader {
			w.Header().Set("Access-Control-Allow-Origin", config.CorsWhitelist)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		next(w, r)
	}
}

func enrichRequestContext(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next(w, r)
			return
		}

		split := strings.Split(authHeader, " ")
		if len(split) != 2 || split[0] != "Bearer" {
			next(w, r)
			return
		}
		next(w, r.Clone(context.WithValue(r.Context(), ContextKeySessionToken, split[1])))
	}
}

func verifyRole(role string, next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionToken := r.Context().Value(ContextKeySessionToken)
		if sessionToken == nil {
			respondUnauthorized(w)
			return
		}

		s, err := sessionStore.Get(sessionToken.(string))
		if err != nil || !session.IsValid(&s) {
			respondUnauthorized(w)
			return
		}

		if !users.CheckRole(s.Role, role) {
			respondForbidden(w)
			return
		}
		next(w, r)
	}
}

func getItems(w http.ResponseWriter, r *http.Request) {
	allItems, err := items.GetAllItems(r.Context(), database)

	if err != nil {
		logAndRespondWithInternalError(w, "Error while retrieving items from database:", err)
		return
	}
	respondWithJson(w, allItems)
}

func respondWithJson(w http.ResponseWriter, response any) {
	resp, err := json.Marshal(response)

	if err != nil {
		logAndRespondWithInternalError(w, "Error while creating json response:", err)
		return
	}

	activateJsonResponse(w)
	_, err = w.Write(resp)
}

func logAndRespondWithInternalError(w http.ResponseWriter, errMessage string, err error) {
	log.Println(errMessage, err)
	w.WriteHeader(http.StatusInternalServerError)
}

func respondBadRequest(w http.ResponseWriter, errMessage string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(errMessage))
}

func respondUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
}

func respondForbidden(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
}

func activateJsonResponse(w http.ResponseWriter) {
	w.Header().Set("content-type", "application/json")
}

func logAndCreateError(message string, err error) error {
	log.Println(message, err)
	return errors.New(message)
}

type Validatable interface {
	Validate() error
}

func readValidJsonBody[T Validatable](r *http.Request) (T, error) {
	var req T

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	var req addItemRequest
	err = json.Unmarshal(rawBody, &req)

	if err != nil {
		return req, logAndCreateError("error unmarshalling json request body", err)
	}

	err = req.Validate()
	return req, err
}

func addItem(w http.ResponseWriter, r *http.Request) {
	reqPointer, err := readValidJsonBody[*addItemRequest](r)

	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	req := *reqPointer

	_, err = items.GetItemByName(r.Context(), req.Name, database)

	if err != nil {
		respondBadRequest(w, "Item already exists!")
		return
	}

	item := items.Item{
		Name:    req.Name,
		Price:   req.Price,
		Image:   req.Image,
		Amount:  req.Amount,
		Id:      uuid.New().String(),
		Barcode: req.Barcode,
	}
	err = items.InsertNewItem(r.Context(), &item, database)

	if err != nil {
		logAndRespondWithInternalError(w, "Error while inserting new item", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	respondWithJson(w, item)
}

func updateItem(w http.ResponseWriter, r *http.Request) {
	reqPointer, err := readValidJsonBody[*updateItemRequest](r)

	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	req := *reqPointer

	item, err := items.GetItemByName(r.Context(), req.Name, database)
	if err == nil && item.Id != req.Id {
		respondBadRequest(w, "an item with this name already exits")
		return
	}
	err = items.UpdateItem(r.Context(), &items.Item{
		Name:    req.Name,
		Price:   req.Price,
		Image:   req.Image,
		Amount:  req.Amount,
		Id:      req.Id,
		Barcode: req.Barcode,
	}, database)

	if err != nil {
		logAndRespondWithInternalError(w, "Error while updating item", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	allUsers, err := users.GetAllUsers(r.Context(), database)
	if err != nil {
		logAndRespondWithInternalError(w, "Error while retrieving users from database:", err)
		return
	}
	respondWithJson(w, allUsers)
}

func getUsersWithNoneAuth(w http.ResponseWriter, r *http.Request) {
	userNames, err := users.GetUsernamesWithNoneAuth(r.Context(), database)
	if err != nil {
		logAndRespondWithInternalError(w, "Error getting list of users with none auth:", err)
		return
	}
	respondWithJson(w, userNames)
}

func registerWithPassword(w http.ResponseWriter, r *http.Request) {
	reqPointer, err := readValidJsonBody[*passwordRegistrationRequest](r)

	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	defer r.Body.Close()

	req := *reqPointer

	_, err = users.GetUserForUsername(r.Context(), req.Username, database)
	if err == nil {
		respondBadRequest(w, "Username already taken")
		return
	}
	user := users.User{
		Id:       uuid.New().String(),
		Username: req.Username,
		Email:    req.Email,
		Role:     "user",
		Credit:   0,
	}
	err = users.AddUser(r.Context(), user, database)
	if err != nil {
		logAndRespondWithInternalError(w, "Error while adding user to database:", err)
		return
	}
	auth := users.AuthenticationData{
		User: user.Id,
		Type: "password",
		Data: users.CalculatePasswordHash(req.Password),
	}
	err = users.AddAuthentication(r.Context(), auth, database)
	if err != nil {
		logAndRespondWithInternalError(w, "Error saving auth:", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func addAuthMethod(w http.ResponseWriter, r *http.Request) {
	token := r.Context().Value(ContextKeySessionToken)
	if token == nil {
		respondUnauthorized(w)
		return
	}

	sess, err := sessionStore.Get(token.(string))
	if err != nil || sess.AuthBackend != "password" {
		respondUnauthorized(w)
		return
	}

	reqPointer, err := readValidJsonBody[*addAuthMethodRequest](r)

	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	req := *reqPointer

	data, _ := hex.DecodeString(req.Data) // already checked in the validate function
	auth := users.AuthenticationData{
		User: sess.UserId,
		Type: req.Method,
		Data: data,
	}
	err = users.AddAuthentication(r.Context(), auth, database)
	if err != nil {
		logAndRespondWithInternalError(w, "Error saving auth data:", err)
		return
	}
}

func loginWithPassword(w http.ResponseWriter, r *http.Request) {
	reqPointer, err := readValidJsonBody[*passwordLoginRequest](r)

	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	defer r.Body.Close()

	req := *reqPointer
	user, err := users.GetUserForUsername(r.Context(), req.Username, database)
	if err != nil {
		respondForbidden(w)
		return
	}
	auth, err := users.GetAuthForUser(r.Context(), user.Id, "password", database)

	if err != nil {
		logAndRespondWithInternalError(w, "Could not get auth data", err)
	}

	if !users.VerifyPasswordHash(auth.Data, req.Password) {
		respondForbidden(w)
		return
	}
	sess := session.CreateSession(user.Id, user.Role, auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)
	respondWithJson(w,
		loginResponse{
			Token:      sess.Id,
			ValidUntil: sess.NotValidAfter,
		})
}

func loginCash(w http.ResponseWriter, r *http.Request) {
	user, err := users.GetUserForId(r.Context(), users.CASH_USER_ID, database)
	if err != nil {
		logAndRespondWithInternalError(w, "error logging in with cash user:", err)
		return
	}

	sess := session.CreateSession(user.Id, "user", "cash", config.SessionLifetime)
	respondWithJson(w, loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	})
	sessionStore.Store(sess)
}

func loginNone(w http.ResponseWriter, r *http.Request) {
	reqPointer, err := readValidJsonBody[*noneLoginRequest](r)

	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	defer r.Body.Close()

	req := *reqPointer

	user, err := users.GetUserForUsername(r.Context(), req.Username, database)
	if err != nil {
		respondForbidden(w)
		return
	}
	auth, err := users.GetAuthForUser(r.Context(), user.Id, "none", database)

	if err != nil {
		logAndRespondWithInternalError(w, "Could not get auth data", err)
	}

	sess := session.CreateSession(user.Id, "user", auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)
	respondWithJson(w, loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	})
}

func loginNFC(w http.ResponseWriter, r *http.Request) {
	reqPointer, err := readValidJsonBody[*nfcLoginRequest](r)

	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	defer r.Body.Close()

	req := *reqPointer
	token, err := hex.DecodeString(req.Token)
	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	user, err := users.GetUserForNFCToken(r.Context(), token, database)
	if err != nil {
		respondForbidden(w)
		return
	}
	auth, err := users.GetAuthForUser(r.Context(), user.Id, "nfc", database)

	if err != nil {
		logAndRespondWithInternalError(w, "Could not get auth data", err)
	}

	sess := session.CreateSession(user.Id, "user", auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)
	respondWithJson(w, loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	})
}

func logout(w http.ResponseWriter, r *http.Request) {
	token := r.Context().Value(ContextKeySessionToken)
	if token == nil {
		// no session associated with the request, just return gracefully
		w.WriteHeader(http.StatusOK)
		return
	}
	sessionStore.Delete(token.(string))
}

func buyItem(w http.ResponseWriter, r *http.Request) {
	sessionToken := r.Context().Value(ContextKeySessionToken)
	if sessionToken == nil {
		respondUnauthorized(w)
		return
	}
	s, err := sessionStore.Get(sessionToken.(string))
	if err != nil || !session.IsValid(&s) {
		respondUnauthorized(w)
		return
	}

	reqPointer, err := readValidJsonBody[*buyItemRequest](r)
	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	req := *reqPointer

	item, err := items.GetItemById(r.Context(), req.ItemId, database)
	if err != nil {
		log.Println("error getting item:", err)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("no such item"))
		return
	}
	user, err := users.GetUserForId(r.Context(), s.UserId, database)
	if err != nil {
		logAndRespondWithInternalError(w, "error getting user from session:", err)
		return
	}
	err = transactions.MakeTransaction(r.Context(), &user, &item, req.Amount, s.AuthBackend, database)
	if err != nil {
		log.Println("error while performing transaction", err)
		respondBadRequest(w, err.Error())
		return
	}
}

func getTransactions(w http.ResponseWriter, r *http.Request) {
	since := int64(0)
	until := time.Now().Unix()
	if r.URL.Query().Has("since") {
		since, _ = strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	}
	if r.URL.Query().Has("until") {
		until, _ = strconv.ParseInt(r.URL.Query().Get("until"), 10, 64)
	}
	transac, err := transactions.GetTransactionsSince(r.Context(), since, until, database)
	if err != nil {
		logAndRespondWithInternalError(w, "error while retrieving all transactions:", err)
		return
	}

	respondWithJson(w, transac)
}

func getItem(w http.ResponseWriter, r *http.Request) {
	idString := strings.TrimPrefix(r.URL.Path, "/items/")
	id, err := uuid.Parse(idString)
	if err != nil {
		respondBadRequest(w, "invalid item id, uuid expected")
		return
	}
	item, err := items.GetItemById(r.Context(), id.String(), database)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	respondWithJson(w, item)
}

func getItemByBarcode(w http.ResponseWriter, r *http.Request) {
	barcodeString := strings.TrimPrefix(r.URL.Path, "/items/barcode/")
	if !regexp.MustCompile("^[0-9]+$").MatchString(barcodeString) {
		respondBadRequest(w, "invalid item barcode")
		return
	}
	item, err := items.GetItemByBarcode(r.Context(), barcodeString, database)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	respondWithJson(w, item)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	idString := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := uuid.Parse(idString)
	if err != nil {
		respondBadRequest(w, "invalid user id, uuid expected")
		return
	}
	user, err := users.GetUserForId(r.Context(), id.String(), database)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	respondWithJson(w, user)
}

func changeCredit(w http.ResponseWriter, r *http.Request) {
	token := r.Context().Value(ContextKeySessionToken)
	if token == nil {
		respondUnauthorized(w)
		return
	}
	sess, err := sessionStore.Get(token.(string))
	if err != nil || sess.AuthBackend != "password" {
		respondUnauthorized(w)
		return
	}
	user, err := users.GetUserForId(r.Context(), sess.UserId, database)
	if err != nil {
		logAndRespondWithInternalError(w, "Error getting user:", err)
		return
	}
	reqPointer, err := readValidJsonBody[*changeCreditRequest](r)
	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	req := *reqPointer

	if user.Credit+req.Diff < 0 {
		respondBadRequest(w, "lending money is not allowed")
		return
	}
	user.Credit += req.Diff
	err = users.UpdateUser(r.Context(), &user, database)
	if err != nil {
		logAndRespondWithInternalError(w, "Error updating user in database:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	reqPointer, err := readValidJsonBody[*requestPasswordResetRequest](r)
	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	req := *reqPointer

	// doing things async, so response timing is not affected by the process.
	go func() {
		err := users.SendPasswordResetMail(req.Username, database)
		if err != nil {
			log.Println("Error while trying to send password reset mail:", err)
		}
	}()
	w.WriteHeader(http.StatusOK)
	return
}

func resetPassword(w http.ResponseWriter, r *http.Request) {
	reqPointer, err := readValidJsonBody[*resetPasswordRequest](r)
	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	req := *reqPointer
	err = users.ResetPassword(r.Context(), req.Token, req.Password, database)
	if err != nil {
		log.Println(w, "Error resetting password:", err)
		respondBadRequest(w, "Error resetting password")
		return
	}
	w.WriteHeader(http.StatusCreated)
	return
}

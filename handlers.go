package main

import (
	"encoding/hex"
	"github.com/Port39/go-drink/domain_errors"
	"github.com/Port39/go-drink/items"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/transactions"
	"github.com/Port39/go-drink/users"
	"github.com/google/uuid"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func getItems(r *http.Request) any {
	allItems, err := items.GetAllItems(r.Context(), database)

	if err != nil {
		log.Println("Error while retrieving items from database:", err)
		return domain_errors.InternalServerError
	}

	return allItems
}

func addItem(r *http.Request) any {
	reqPointer, err := readValidJsonBody[*addItemRequest](r)

	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}

	req := *reqPointer

	_, err = items.GetItemByName(r.Context(), req.Name, database)

	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, "Item already exists!")
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
		log.Println("Error while inserting new item", err)
		return domain_errors.InternalServerError
	}

	// FIXME regression 200 instead of 201
	return item
}

func updateItem(r *http.Request) any {
	reqPointer, err := readValidJsonBody[*updateItemRequest](r)

	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}

	req := *reqPointer

	item, err := items.GetItemByName(r.Context(), req.Name, database)
	if err == nil && item.Id != req.Id {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, "an item with this name already exits")
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
		log.Println("Error while updating item", err)
		return domain_errors.InternalServerError
	}

	return item
}

func getUsers(r *http.Request) any {
	allUsers, err := users.GetAllUsers(r.Context(), database)
	if err != nil {
		log.Println("Error while retrieving users from database:", err)
		return domain_errors.InternalServerError
	}
	return allUsers
}

func getUsersWithNoneAuth(r *http.Request) any {
	userNames, err := users.GetUsernamesWithNoneAuth(r.Context(), database)
	if err != nil {
		log.Println("Error getting list of users with none auth:", err)
		return domain_errors.InternalServerError
	}
	return userNames
}

func registerWithPassword(r *http.Request) any {
	reqPointer, err := readValidJsonBody[*passwordRegistrationRequest](r)

	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	defer r.Body.Close()

	req := *reqPointer

	_, err = users.GetUserForUsername(r.Context(), req.Username, database)
	if err == nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, "Username already taken")
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
		log.Println("Error while adding user to database:", err)
		return domain_errors.InternalServerError
	}

	auth := users.AuthenticationData{
		User: user.Id,
		Type: "password",
		Data: users.CalculatePasswordHash(req.Password),
	}

	err = users.AddAuthentication(r.Context(), auth, database)

	if err != nil {
		log.Println("Error saving auth:", err)
		return domain_errors.InternalServerError
	}

	return nil
}

func addAuthMethod(r *http.Request) any {
	token := r.Context().Value(ContextKeySessionToken)
	if token == nil {
		return domain_errors.Unauthorized
	}

	sess, err := sessionStore.Get(token.(string))
	if err != nil || sess.AuthBackend != "password" {
		return domain_errors.Unauthorized
	}

	reqPointer, err := readValidJsonBody[*addAuthMethodRequest](r)

	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
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
		log.Println("Error saving auth data:", err)
		return domain_errors.InternalServerError
	}

	// FIXME regression 200 instead of 201
	return nil
}

func loginWithPassword(r *http.Request) any {
	reqPointer, err := readValidJsonBody[*passwordLoginRequest](r)

	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	defer r.Body.Close()

	req := *reqPointer
	user, err := users.GetUserForUsername(r.Context(), req.Username, database)
	if err != nil {
		return domain_errors.Forbidden
	}
	auth, err := users.GetAuthForUser(r.Context(), user.Id, "password", database)

	if err != nil {
		log.Println("Could not get auth data", err)
		return domain_errors.InternalServerError
	}

	if !users.VerifyPasswordHash(auth.Data, req.Password) {
		return domain_errors.Forbidden
	}
	sess := session.CreateSession(user.Id, user.Role, auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)

	return loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	}
}

func loginCash(r *http.Request) any {
	user, err := users.GetUserForId(r.Context(), users.CASH_USER_ID, database)
	if err != nil {
		log.Println("error logging in with cash user:", err)
		return domain_errors.InternalServerError
	}

	sess := session.CreateSession(user.Id, "user", "cash", config.SessionLifetime)
	sessionStore.Store(sess)

	return loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	}
}

func loginNone(r *http.Request) any {
	reqPointer, err := readValidJsonBody[*noneLoginRequest](r)

	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	defer r.Body.Close()

	req := *reqPointer

	user, err := users.GetUserForUsername(r.Context(), req.Username, database)
	if err != nil {
		return domain_errors.Forbidden
	}
	auth, err := users.GetAuthForUser(r.Context(), user.Id, "none", database)

	if err != nil {
		log.Println("Could not get auth data", err)
		return domain_errors.InternalServerError
	}

	sess := session.CreateSession(user.Id, "user", auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)
	return loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	}
}

func loginNFC(r *http.Request) any {
	reqPointer, err := readValidJsonBody[*nfcLoginRequest](r)

	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	defer r.Body.Close()

	req := *reqPointer
	token, err := hex.DecodeString(req.Token)
	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	user, err := users.GetUserForNFCToken(r.Context(), token, database)
	if err != nil {
		return domain_errors.Forbidden
	}
	auth, err := users.GetAuthForUser(r.Context(), user.Id, "nfc", database)

	if err != nil {
		log.Println("Could not get auth data", err)
		return domain_errors.InternalServerError
	}

	sess := session.CreateSession(user.Id, "user", auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)
	return loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	}
}

func logout(r *http.Request) any {
	token := r.Context().Value(ContextKeySessionToken)
	if token == nil {
		// no session associated with the request, just return gracefully
		return nil
	}
	sessionStore.Delete(token.(string))
	return nil
}

func buyItem(r *http.Request) any {
	sessionToken := r.Context().Value(ContextKeySessionToken)
	if sessionToken == nil {
		return domain_errors.Unauthorized
	}
	s, err := sessionStore.Get(sessionToken.(string))
	if err != nil || !session.IsValid(&s) {
		return domain_errors.Unauthorized
	}

	reqPointer, err := readValidJsonBody[*buyItemRequest](r)
	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	req := *reqPointer

	item, err := items.GetItemById(r.Context(), req.ItemId, database)
	if err != nil {
		log.Println("error getting item:", err)
		return domain_errors.NotFound
	}
	user, err := users.GetUserForId(r.Context(), s.UserId, database)
	if err != nil {
		log.Println("error getting user from session:", err)
		return domain_errors.InternalServerError
	}
	err = transactions.MakeTransaction(r.Context(), &user, &item, req.Amount, s.AuthBackend, database)
	if err != nil {
		log.Println("error while performing transaction", err)
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	return nil
}

func getTransactions(r *http.Request) any {
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
		log.Println("error while retrieving all transactions:", err)
		return domain_errors.InternalServerError
	}

	return transac
}

func getItem(r *http.Request) any {
	idString := strings.TrimPrefix(r.URL.Path, "/items/")
	id, err := uuid.Parse(idString)
	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, "invalid item id, uuid expected")
	}
	item, err := items.GetItemById(r.Context(), id.String(), database)
	if err != nil {
		return domain_errors.NotFound
	}

	return item
}

func getItemByBarcode(r *http.Request) any {
	barcodeString := strings.TrimPrefix(r.URL.Path, "/items/barcode/")
	if !regexp.MustCompile("^[0-9]+$").MatchString(barcodeString) {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, "invalid item barcode")
	}
	item, err := items.GetItemByBarcode(r.Context(), barcodeString, database)
	if err != nil {
		return domain_errors.NotFound
	}

	return item
}

func getUser(r *http.Request) any {
	idString := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := uuid.Parse(idString)
	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, "invalid user id, uuid expected")
	}
	user, err := users.GetUserForId(r.Context(), id.String(), database)
	if err != nil {
		return domain_errors.NotFound
	}
	return user
}

func changeCredit(r *http.Request) any {
	token := r.Context().Value(ContextKeySessionToken)
	if token == nil {
		return domain_errors.Unauthorized
	}
	sess, err := sessionStore.Get(token.(string))
	if err != nil || sess.AuthBackend != "password" {
		return domain_errors.Unauthorized
	}
	user, err := users.GetUserForId(r.Context(), sess.UserId, database)
	if err != nil {
		log.Println("Error getting user:", err)
		return domain_errors.InternalServerError
	}
	reqPointer, err := readValidJsonBody[*changeCreditRequest](r)
	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	req := *reqPointer

	if user.Credit+req.Diff < 0 {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, "lending money is not allowed")
	}
	user.Credit += req.Diff
	err = users.UpdateUser(r.Context(), &user, database)
	if err != nil {
		log.Println("Error updating user in database:", err)
		return domain_errors.InternalServerError
	}

	return nil
}

func requestPasswordReset(r *http.Request) any {
	reqPointer, err := readValidJsonBody[*requestPasswordResetRequest](r)
	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	req := *reqPointer

	// doing things async, so response timing is not affected by the process.
	go func() {
		err := users.SendPasswordResetMail(req.Username, database)
		if err != nil {
			log.Println("Error while trying to send password reset mail:", err)
		}
	}()
	return nil
}

func resetPassword(r *http.Request) any {
	reqPointer, err := readValidJsonBody[*resetPasswordRequest](r)
	if err != nil {
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, err.Error())
	}
	req := *reqPointer
	err = users.ResetPassword(r.Context(), req.Token, req.Password, database)
	if err != nil {
		log.Println("Error resetting password:", err)
		return domain_errors.ForStatusAndDetail(http.StatusBadRequest, "Error resetting password")
	}

	return nil
}

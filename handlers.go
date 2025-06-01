package main

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Port39/go-drink/domain_errors"
	"github.com/Port39/go-drink/handlehttp"
	contenttype "github.com/Port39/go-drink/handlehttp/content-type"
	"github.com/Port39/go-drink/items"
	"github.com/Port39/go-drink/session"
	"github.com/Port39/go-drink/transactions"
	"github.com/Port39/go-drink/users"
	"github.com/google/uuid"
)

func errorWithContext(ctx context.Context, status int) (context.Context, any) {
	return handlehttp.ContextWithStatus(ctx, status), domain_errors.ForStatus(status)
}
func errorWithContextAndDetail(ctx context.Context, status int, detail string) (context.Context, any) {
	return handlehttp.ContextWithStatus(ctx, status), domain_errors.ForStatusAndDetail(status, detail)
}

var getItems handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	allItems, err := items.GetAllItems(r.Context(), database)

	if err != nil {
		log.Println("Error while retrieving items from database:", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), allItems
}

var addItem handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	req, err := handlehttp.ReadValidBody[addItemRequest](r)

	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}

	_, err = items.GetItemByName(r.Context(), req.Name, database)
	log.Println(err)

	if err == nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, "Item already exists!")
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
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	return handlehttp.ContextWithStatus(r.Context(), http.StatusCreated), item
}

var updateItem handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	req, err := handlehttp.ReadValidBody[updateItemRequest](r)

	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}

	item, err := items.GetItemByName(r.Context(), req.Name, database)
	if err == nil && item.Id != req.Id {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, "an item with this name already exits")
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
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), item
}

var getUsers handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	allUsers, err := users.GetAllUsers(r.Context(), database)
	if err != nil {
		log.Println("Error while retrieving users from database:", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}
	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), allUsers
}

var getUsersWithNoneAuth handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	userNames, err := users.GetUsernamesWithNoneAuth(r.Context(), database)
	if err != nil {
		log.Println("Error getting list of users with none auth:", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}
	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), userNames
}

var registerWithPassword handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	req, err := handlehttp.ReadValidBody[passwordRegistrationRequest](r)

	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}
	defer r.Body.Close()

	_, err = users.GetUserForUsername(r.Context(), req.Username, database)
	if err == nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, "Username already taken")
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
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	auth := users.AuthenticationData{
		User: user.Id,
		Type: "password",
		Data: users.CalculatePasswordHash(req.Password),
	}

	err = users.AddAuthentication(r.Context(), auth, database)

	if err != nil {
		log.Println("Error saving auth:", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	sess := session.CreateSession(user.Id, user.Role, auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)

	ctx := handlehttp.ContextWithSession(r.Context(), sess)
	ctx = handlehttp.ContextWithStatus(ctx, http.StatusCreated)

	return ctx, user
}

var addAuthMethod handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	token, hasToken := handlehttp.ContextGetSessionToken(r.Context())
	if !hasToken {
		return errorWithContext(r.Context(), http.StatusUnauthorized)
	}

	sess, err := sessionStore.Get(token)
	if err != nil || sess.AuthBackend != "password" {
		return errorWithContext(r.Context(), http.StatusUnauthorized)
	}

	req, err := handlehttp.ReadValidBody[addAuthMethodRequest](r)

	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}

	data, _ := hex.DecodeString(req.Data) // already checked in the validate function
	auth := users.AuthenticationData{
		User: sess.UserId,
		Type: req.Method,
		Data: data,
	}
	err = users.AddAuthentication(r.Context(), auth, database)
	if err != nil {
		log.Println("Error saving auth data:", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	return handlehttp.ContextWithStatus(r.Context(), http.StatusCreated), nil
}

var loginWithPassword handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	req, err := handlehttp.ReadValidBody[passwordLoginRequest](r)

	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}
	defer r.Body.Close()

	user, err := users.GetUserForUsername(r.Context(), req.Username, database)
	if err != nil {
		return errorWithContext(r.Context(), http.StatusForbidden)
	}
	auth, err := users.GetAuthForUser(r.Context(), user.Id, "password", database)

	if err != nil {
		log.Println("Could not get auth data", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	if !users.VerifyPasswordHash(auth.Data, req.Password) {
		return errorWithContext(r.Context(), http.StatusForbidden)
	}
	sess := session.CreateSession(user.Id, user.Role, auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)

	ctx := handlehttp.ContextWithSession(r.Context(), sess)
	ctx = handlehttp.ContextWithStatus(ctx, http.StatusOK)

	return ctx, loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	}
}

var loginCash handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	user, err := users.GetUserForId(r.Context(), users.CashUserId, database)
	if err != nil {
		log.Println("error logging in with cash user:", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	sess := session.CreateSession(user.Id, "user", "cash", config.SessionLifetime)
	sessionStore.Store(sess)

	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	}
}

var loginNone handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	req, err := handlehttp.ReadValidBody[noneLoginRequest](r)

	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}
	defer r.Body.Close()

	user, err := users.GetUserForUsername(r.Context(), req.Username, database)
	if err != nil {
		return errorWithContext(r.Context(), http.StatusForbidden)
	}
	auth, err := users.GetAuthForUser(r.Context(), user.Id, "none", database)

	if err != nil {
		log.Println("Could not get auth data", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	sess := session.CreateSession(user.Id, "user", auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)
	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	}
}

var loginNFC handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	req, err := handlehttp.ReadValidBody[nfcLoginRequest](r)

	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}
	defer r.Body.Close()

	token, err := hex.DecodeString(req.Token)
	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}
	user, err := users.GetUserForNFCToken(r.Context(), token, database)
	if err != nil {
		return errorWithContext(r.Context(), http.StatusForbidden)
	}
	auth, err := users.GetAuthForUser(r.Context(), user.Id, "nfc", database)

	if err != nil {
		log.Println("Could not get auth data", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	sess := session.CreateSession(user.Id, "user", auth.Type, config.SessionLifetime)
	sessionStore.Store(sess)
	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	}
}

var logout handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	token, hasToken := handlehttp.ContextGetSessionToken(r.Context())
	if hasToken {
		sessionStore.Delete(token)
	}
	mediatype, _, err := contenttype.GetAcceptableMediaType(r, []contenttype.MediaType{handlehttp.Html, handlehttp.Json})

	var result int

	if err == nil && mediatype.Matches(handlehttp.Html) {
		result = http.StatusOK
	} else {
		result = http.StatusCreated
	}

	ctx := handlehttp.ContextWithoutSession(r.Context())
	ctx = handlehttp.ContextWithStatus(ctx, result)

	return ctx, nil
}

var buyItem handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	token, hasToken := handlehttp.ContextGetSessionToken(r.Context())
	if !hasToken {
		return errorWithContext(r.Context(), http.StatusUnauthorized)
	}
	s, err := sessionStore.Get(token)
	if err != nil || !session.IsValid(&s) {
		return errorWithContext(r.Context(), http.StatusUnauthorized)
	}

	req, err := handlehttp.ReadValidBody[buyItemRequest](r)
	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}

	item, err := items.GetItemById(r.Context(), req.ItemId, database)
	if err != nil {
		log.Println("error getting item:", err)
		return errorWithContext(r.Context(), http.StatusNotFound)
	}
	user, err := users.GetUserForId(r.Context(), s.UserId, database)
	if err != nil {
		log.Println("error getting user from session:", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}
	err = transactions.MakeTransaction(r.Context(), &user, &item, req.Amount, s.AuthBackend, database)
	if err != nil {
		log.Println("error while performing transaction", err)
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}
	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), nil
}

var getTransactions handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
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
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), transac
}

var getItem handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	idString := strings.TrimPrefix(r.URL.Path, "/items/")
	id, err := uuid.Parse(idString)
	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, "invalid item id, uuid expected")
	}
	item, err := items.GetItemById(r.Context(), id.String(), database)
	if err != nil {
		return errorWithContext(r.Context(), http.StatusNotFound)
	}

	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), item
}

var getItemByBarcode handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	barcodeString := strings.TrimPrefix(r.URL.Path, "/items/barcode/")
	if !regexp.MustCompile("^[0-9]+$").MatchString(barcodeString) {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, "invalid item barcode")
	}
	item, err := items.GetItemByBarcode(r.Context(), barcodeString, database)
	if err != nil {
		return errorWithContext(r.Context(), http.StatusNotFound)
	}

	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), item
}

var getUser handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	idString := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := uuid.Parse(idString)
	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, "invalid user id, uuid expected")
	}
	user, err := users.GetUserForId(r.Context(), id.String(), database)
	if err != nil {
		return errorWithContext(r.Context(), http.StatusNotFound)
	}
	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), user
}

var changeCredit handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	token, hasToken := handlehttp.ContextGetSessionToken(r.Context())
	if !hasToken {
		return errorWithContext(r.Context(), http.StatusUnauthorized)
	}
	sess, err := sessionStore.Get(token)
	if err != nil || sess.AuthBackend != "password" {
		return errorWithContext(r.Context(), http.StatusUnauthorized)
	}
	user, err := users.GetUserForId(r.Context(), sess.UserId, database)
	if err != nil {
		log.Println("Error getting user:", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}
	req, err := handlehttp.ReadValidBody[changeCreditRequest](r)
	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}

	if user.Credit+req.Diff < 0 {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, "lending money is not allowed")
	}
	user.Credit += req.Diff
	err = users.UpdateUser(r.Context(), &user, database)
	if err != nil {
		log.Println("Error updating user in database:", err)
		return errorWithContext(r.Context(), http.StatusInternalServerError)
	}

	return handlehttp.ContextWithStatus(r.Context(), http.StatusOK), nil
}

var requestPasswordReset handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	req, err := handlehttp.ReadValidBody[requestPasswordResetRequest](r)
	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}

	// doing things async, so response timing is not affected by the process.
	go func() {
		err := users.SendPasswordResetMail(req.Username, database)
		if err != nil {
			log.Println("Error while trying to send password reset mail:", err)
		}
	}()
	return handlehttp.ContextWithStatus(r.Context(), http.StatusNoContent), nil
}

var resetPassword handlehttp.RequestHandler = func(r *http.Request) (context.Context, any) {
	req, err := handlehttp.ReadValidBody[resetPasswordRequest](r)
	if err != nil {
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, err.Error())
	}
	err = users.ResetPassword(r.Context(), req.Token, req.Password, database)
	if err != nil {
		log.Println("Error resetting password:", err)
		return errorWithContextAndDetail(r.Context(), http.StatusBadRequest, "Error resetting password")
	}

	return handlehttp.ContextWithStatus(r.Context(), http.StatusCreated), nil
}

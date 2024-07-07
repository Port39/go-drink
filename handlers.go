package main

import (
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
	"strings"
)

func getSessionToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	split := strings.Split(authHeader, " ")
	if len(split) != 2 || split[0] != "Bearer" {
		return "", errors.New("invalid token format")
	}
	return split[1], nil
}

func verifyRole(role string, next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionToken, err := getSessionToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		s, err := sessionStore.Get(sessionToken)
		if err != nil || !session.IsValid(&s) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !users.CheckRole(s.Role, role) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func getItems(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "application/json")
	aLlItems, err := items.GetALlItems(database)
	if err != nil {
		log.Println("Error while retrieving items from database:", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("[]"))
		return
	}
	resp, err := json.Marshal(aLlItems)
	if err != nil {
		log.Println("Error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("[]"))
		return
	}
	_, err = w.Write(resp)
}

func addItem(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var req addItemRequest
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = req.Validate()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	_, err = items.GetItemByName(req.Name, database)
	if err == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Item already exists!"))
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
	err = items.InsertNewItem(&item, database)
	if err != nil {
		log.Println("error inserting new Item:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp, err := json.Marshal(item)
	if err != nil {
		log.Println("Error creating response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(resp)
}

func updateItem(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var req updateItemRequest
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = req.Validate()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = req.Validate()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	item, err := items.GetItemByName(req.Name, database)
	if err == nil && item.Id != req.Id {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("an item with this name already exits"))
		return
	}
	err = items.UpdateItem(&items.Item{
		Name:    req.Name,
		Price:   req.Price,
		Image:   req.Image,
		Amount:  req.Amount,
		Id:      req.Id,
		Barcode: req.Barcode,
	}, database)
	if err != nil {
		log.Println("Error saving item:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func getUsers(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "application/json")
	allUsers, err := users.GetAllUsers(database)
	if err != nil {
		log.Println("Error while retrieving users from database:", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("[]"))
		return
	}
	resp, err := json.Marshal(allUsers)
	if err != nil {
		log.Println("Error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("[]"))
		return
	}
	_, err = w.Write(resp)
}

func getUsersWithNoneAuth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "application/json")
	usernames, err := users.GetUsernamesWithNoneAuth(database)
	if err != nil {
		log.Println("Error getting list of users with none auth:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[]"))
		return
	}
	data, err := json.Marshal(usernames)
	if err != nil {
		log.Println("Error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[]"))
		return
	}
	_, err = w.Write(data)
}

func registerWithPassword(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var req passwordRegistrationRequest
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = req.Validate()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	_, err = users.GetUserForUsername(req.Username, database)
	if err == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("user already exists"))
		return
	}
	user := users.User{
		Id:       uuid.New().String(),
		Username: req.Username,
		Email:    req.Email,
		Role:     "user",
		Credit:   0,
	}
	err = users.AddUser(user, database)
	if err != nil {
		log.Println("Error saving user:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	auth := users.AuthenticationData{
		User: user.Id,
		Type: "password",
		Data: users.CalculatePasswordHash(req.Password),
	}
	err = users.AddAuthentication(auth, database)
	if err != nil {
		log.Println("Error saving auth:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func addAuthMethod(w http.ResponseWriter, r *http.Request) {
	token, err := getSessionToken(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	sess, err := sessionStore.Get(token)
	if err != nil || sess.AuthBackend != "password" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	rawBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var req addAuthMethodRequest
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = req.Validate()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	data, _ := hex.DecodeString(req.Data) // already checked in the validate function
	auth := users.AuthenticationData{
		User: sess.UserId,
		Type: req.Method,
		Data: data,
	}
	err = users.AddAuthentication(auth, database)
	if err != nil {
		log.Println("Error saving auth data", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func loginWithPassword(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var req passwordLoginRequest
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	user, err := users.GetUserForUsername(req.Username, database)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	auth, err := users.GetAuthForUser(user.Id, "password", database)
	if !users.VerifyPasswordHash(auth.Data, req.Password) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	sess := session.CreateSession(user.Id, user.Role, auth.Type, config.SessionLifetime)
	resp, err := json.Marshal(loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	})
	if err != nil {
		log.Println("Error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionStore.Store(sess)
	w.Header().Set("content-type", "application/json")
	_, err = w.Write(resp)
}

func loginCash(w http.ResponseWriter, _ *http.Request) {
	user, err := users.GetUserForId(users.CASH_USER_ID, database)
	if err != nil {
		log.Println("error logging in with cash user:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sess := session.CreateSession(user.Id, "user", "cash", config.SessionLifetime)
	resp, err := json.Marshal(loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	})
	if err != nil {
		log.Println("Error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionStore.Store(sess)
	w.Header().Set("content-type", "application/json")
	_, err = w.Write(resp)
}

func loginNone(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var req noneLoginRequest
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	user, err := users.GetUserForUsername(req.Username, database)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	auth, err := users.GetAuthForUser(user.Id, "none", database)

	sess := session.CreateSession(user.Id, "user", auth.Type, config.SessionLifetime)
	resp, err := json.Marshal(loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	})
	if err != nil {
		log.Println("Error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionStore.Store(sess)
	w.Header().Set("content-type", "application/json")
	_, err = w.Write(resp)
}

func loginNFC(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var req nfcLoginRequest
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	token, err := hex.DecodeString(req.Token)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	user, err := users.GetUserForNFCToken(token, database)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	auth, err := users.GetAuthForUser(user.Id, "nfc", database)

	sess := session.CreateSession(user.Id, "user", auth.Type, config.SessionLifetime)
	resp, err := json.Marshal(loginResponse{
		Token:      sess.Id,
		ValidUntil: sess.NotValidAfter,
	})
	if err != nil {
		log.Println("Error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionStore.Store(sess)
	w.Header().Set("content-type", "application/json")
	_, err = w.Write(resp)
}

func logout(w http.ResponseWriter, r *http.Request) {
	token, err := getSessionToken(r)
	if err != nil {
		// no session associated with the request, just return gracefully
		w.WriteHeader(http.StatusOK)
		return
	}
	sessionStore.Delete(token)
}

func buyItem(w http.ResponseWriter, r *http.Request) {
	sessionToken, err := getSessionToken(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	s, err := sessionStore.Get(sessionToken)
	if err != nil || !session.IsValid(&s) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var req buyItemRequest
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	if err = req.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	item, err := items.GetItemById(req.ItemId, database)
	if err != nil {
		log.Println("error getting item:", err)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("no such item"))
		return
	}
	user, err := users.GetUserForId(s.UserId, database)
	if err != nil {
		log.Println("error getting user from session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = transactions.MakeTransaction(&user, &item, req.Amount, s.AuthBackend, database)
	if err != nil {
		log.Println("error while performing transaction", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
}

func getTransactions(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "application/json")
	transac, err := transactions.GetAllTransactions(database)
	if err != nil {
		log.Println("error while retrieving all transactions:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[]"))
		return
	}
	resp, err := json.Marshal(transac)
	if err != nil {
		log.Println("error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[]"))
		return
	}
	_, err = w.Write(resp)
}

func getItem(w http.ResponseWriter, r *http.Request) {
	idString := strings.TrimPrefix(r.URL.Path, "/items/")
	id, err := uuid.Parse(idString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid item id, uuid expected"))
		return
	}
	item, err := items.GetItemById(id.String(), database)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	resp, err := json.Marshal(item)
	if err != nil {
		log.Println("error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	_, err = w.Write(resp)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	idString := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := uuid.Parse(idString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid user id, uuid expected"))
		return
	}
	user, err := users.GetUserForId(id.String(), database)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	resp, err := json.Marshal(user)
	if err != nil {
		log.Println("error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	_, err = w.Write(resp)
}

func changeCredit(w http.ResponseWriter, r *http.Request) {
	token, err := getSessionToken(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	sess, err := sessionStore.Get(token)
	if err != nil || sess.AuthBackend != "password" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	user, err := users.GetUserForId(sess.UserId, database)
	if err != nil {
		log.Println("Error getting user:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rawBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println("Error reading body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var req changeCreditRequest
	err = json.Unmarshal(rawBody, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	if user.Credit+req.Diff < 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("lending money is not allowed"))
		return
	}
	user.Credit += req.Diff
	err = users.UpdateUser(&user, database)
	if err != nil {
		log.Println("Error updating user in database:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

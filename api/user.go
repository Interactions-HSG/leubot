package api

import (
	"encoding/json"
	"log"
	"net/http"
	"path"
)

// User provides the struct for the user
type User struct {
	Name  string
	Email string
	Token string
}

// UserInfo provides the JSON scheme for User
type UserInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ToUserInfo parses User to UserInfo
func (u *User) ToUserInfo() UserInfo {
	return UserInfo{
		Name:  u.Name,
		Email: u.Email,
	}
}

// NewUser instantiate a user
func NewUser(userInfo *UserInfo) *User {
	return &User{
		Name:  userInfo.Name,
		Email: userInfo.Email,
		Token: GenerateToken(),
	}
}

// UserHandler process the requests on the user
func UserHandler(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "X-API-Key")

	// respond to HEAD or OPTIONS
	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
		return
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getUser(w, r)
	case http.MethodDelete:
		removeUser(w, r)
	case http.MethodPost:
		addUser(w, r)
	}
}

func addUser(w http.ResponseWriter, r *http.Request) {
	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var userInfo UserInfo
	err := decoder.Decode(&userInfo)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypeAddUser,
		Value: []interface{}{userInfo},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeUserAdded: // respond with the added UserInfo
		user, ok := msg.Value[0].(User)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
		}
		log.Printf("[HandlerChannel] UserAdded (name, email, token) = %v, %v, %v", user.Name, user.Email, user.Token)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Location", APIProto+APIHost+APIBasePath+"/user/"+user.Token)
		w.WriteHeader(http.StatusCreated)
	case TypeUserExisted: // there's a user in the system already
		log.Printf("[HandlerChannel] UserExisted, not replacing with (name, email) = %v, %v", userInfo.Name, userInfo.Email)
		w.WriteHeader(http.StatusConflict)
	case TypeInvalidUserInfo: // invalid email
		log.Printf("[HandlerChannel] Invalid UserInfo (name, email) = %v, %v", userInfo.Name, userInfo.Email)
		w.WriteHeader(http.StatusBadRequest)
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func getUser(w http.ResponseWriter, r *http.Request) {
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type: TypeGetUser,
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeCurrentUser: // respond with the current UserInfo
		userInfo, ok := msg.Value[0].(UserInfo)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
		}
		log.Printf("[HandlerChannel] CurrentUser (name, email) = %v, %v", userInfo.Name, userInfo.Email)
		js, err := json.Marshal(userInfo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(js)
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func removeUser(w http.ResponseWriter, r *http.Request) {
	// get the token from the path
	token := path.Base(r.URL.Path)
	HandlerChannel <- HandlerMessage{
		Type:  TypeDeleteUser,
		Value: []interface{}{token},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeUserDeleted: // the user removed
		log.Printf("[HandlerChannel] UserDeleted with token = %v", token)
		w.WriteHeader(http.StatusNoContent)
	case TypeUserNotFound: // no user with the token
		log.Printf("[HandlerChannel] UserNotfound with token = %v", token)
		w.WriteHeader(http.StatusNotFound)
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError)
	}
}

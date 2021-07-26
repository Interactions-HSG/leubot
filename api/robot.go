package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// RobotCommand is a struct for each command
type RobotCommand struct {
	Token string `json:"token"`
	Value uint16 `json:"value"`
}

// PostureCommand is a struct for a posture
type PostureCommand struct {
	Token         string `json:"token"`
	Base          uint16 `json:"base"`
	Shoulder      uint16 `json:"shoulder"`
	Elbow         uint16 `json:"elbow"`
	WristAngle    uint16 `json:"wristAngle"`
	WristRotation uint16 `json:"wristRotation"`
	Gripper       uint16 `json:"gripper"`
	Delta         uint8  `json:"delta"`
}

// PutBase processes the request for Base
func PutBase(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "PUT")

	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var robotCommand RobotCommand
	err := decoder.Decode(&robotCommand)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutBase,
		Value: []interface{}{robotCommand},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Printf("[HandlerChannel] PutBase: %v", robotCommand.Value)
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidCommand: // the invalid value provided
		log.Printf("[HandlerChannel] InvalidCommand: %v", robotCommand.Value)
		w.WriteHeader(http.StatusBadRequest) // 400
	case TypeInvalidToken: // the invalid token provided
		log.Printf("[HandlerChannel] InvalidToken: %v", robotCommand.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

// PutShoulder processes the request for Shoulder
func PutShoulder(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "PUT")

	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var robotCommand RobotCommand
	err := decoder.Decode(&robotCommand)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutShoulder,
		Value: []interface{}{robotCommand},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Printf("[HandlerChannel] PutShoulder: %v", robotCommand.Value)
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidCommand: // the invalid value provided
		log.Printf("[HandlerChannel] InvalidCommand: %v", robotCommand.Value)
		w.WriteHeader(http.StatusBadRequest) // 400
	case TypeInvalidToken: // the invalid token provided
		log.Printf("[HandlerChannel] InvalidToken: %v", robotCommand.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

// PutElbow processes the request for Elbow
func PutElbow(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "PUT")
	
	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var robotCommand RobotCommand
	err := decoder.Decode(&robotCommand)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutElbow,
		Value: []interface{}{robotCommand},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Printf("[HandlerChannel] ElbowRotation: %v", robotCommand.Value)
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidCommand: // the invalid value provided
		log.Printf("[HandlerChannel] InvalidCommand: %v", robotCommand.Value)
		w.WriteHeader(http.StatusBadRequest) // 400
	case TypeInvalidToken: // the invalid token provided
		log.Printf("[HandlerChannel] InvalidToken: %v", robotCommand.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

// PutWristAngle processes the request for WristAngle
func PutWristAngle(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "PUT")
	
	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var robotCommand RobotCommand
	err := decoder.Decode(&robotCommand)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutWristAngle,
		Value: []interface{}{robotCommand},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Printf("[HandlerChannel] WristAngle: %v", robotCommand.Value)
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidCommand: // the invalid value provided
		log.Printf("[HandlerChannel] InvalidCommand: %v", robotCommand.Value)
		w.WriteHeader(http.StatusBadRequest) // 400
	case TypeInvalidToken: // the invalid token provided
		log.Printf("[HandlerChannel] InvalidToken: %v", robotCommand.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

// PutWristRotation processes the request for WristRotation
func PutWristRotation(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "PUT")
	
	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var robotCommand RobotCommand
	err := decoder.Decode(&robotCommand)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutWristRotation,
		Value: []interface{}{robotCommand},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Printf("[HandlerChannel] WristRotation: %v", robotCommand.Value)
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidCommand: // the invalid value provided
		log.Printf("[HandlerChannel] InvalidCommand: %v", robotCommand.Value)
		w.WriteHeader(http.StatusBadRequest) // 400
	case TypeInvalidToken: // the invalid token provided
		log.Printf("[HandlerChannel] InvalidToken: %v", robotCommand.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

// PutGripper processes the request for Gripper
func PutGripper(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "PUT")
	
	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var robotCommand RobotCommand
	err := decoder.Decode(&robotCommand)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutGripper,
		Value: []interface{}{robotCommand},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Printf("[HandlerChannel] Gripper: %v", robotCommand.Value)
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidCommand: // the invalid value provided
		log.Printf("[HandlerChannel] InvalidCommand: %v", robotCommand.Value)
		w.WriteHeader(http.StatusBadRequest) // 400
	case TypeInvalidToken: // the invalid token provided
		log.Printf("[HandlerChannel] InvalidToken: %v", robotCommand.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

// PutPosture sets all the joints at once
func PutPosture(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "PUT")
	
	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var posCom PostureCommand
	err := decoder.Decode(&posCom)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutPosture,
		Value: []interface{}{posCom},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Println("[HandlerChannel] Posture")
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidToken: // the invalid token provided
		log.Printf("[HandlerChannel] InvalidToken: %v", posCom.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

// PutReset processes the request to reset
func PutReset(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "PUT")
	
	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var robotCommand RobotCommand
	err := decoder.Decode(&robotCommand)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutReset,
		Value: []interface{}{robotCommand},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Println("[HandlerChannel] Reset")
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidToken: // the invalid token provided
		log.Printf("[HandlerChannel] InvalidToken: %v", robotCommand.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	default: // something went wrong
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

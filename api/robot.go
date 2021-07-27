package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Interactions-HSG/leubot/armlink"
)

// JointInfo is a struct for each joint
type JointInfo struct {
	Name  string `json:"name"`
	Value uint16 `json:"value"`
}

// RobotCommand is a struct for each command
type RobotCommand struct {
	Token string `json:"token"`
	Value uint16 `json:"value"`
}

// RobotPose stores the rotations of each joint
type RobotPose struct {
	Base          uint16
	Shoulder      uint16
	Elbow         uint16
	WristAngle    uint16
	WristRotation uint16
	Gripper       uint16
}

// BuildArmLinkPacket creates a new ArmLinkPacket
func (rp *RobotPose) BuildArmLinkPacket(delta uint8) *armlink.ArmLinkPacket {
	return armlink.NewArmLinkPacket(rp.Base, rp.Shoulder, rp.Elbow, rp.WristAngle, rp.WristRotation, rp.Gripper, delta, 0, 0)
}

// String returns a string rep for the rp
func (rp *RobotPose) String() string {
	return fmt.Sprintf("Base: %v, Shoulder: %v, Elbow: %v, WristAngle: %v, WristRotation: %v, Gripper: %v", rp.Base, rp.Shoulder, rp.Elbow, rp.WristAngle, rp.WristRotation, rp.Gripper)
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

// RobotHandler process the request to /base
func RobotHandler(w http.ResponseWriter, r *http.Request) {
	// allow CORS here By * or specific origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	// respond to HEAD or OPTIONS
	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
		return
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return
	}

	// process GET and PUT
	switch r.Method {
	case http.MethodGet:
		getState(w, r)
	case http.MethodPut:
		putState(w, r)
	}
}

// getPosture gets the current posture
func getPosture(w http.ResponseWriter, r *http.Request) {
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypeGetPosture,
		Value: []interface{}{},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	// respond with the result
	rp, ok := msg.Value[0].(RobotPose)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
	}
	js, err := json.Marshal(rp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}

// getState gets the current value for each joint
func getState(w http.ResponseWriter, r *http.Request) {
	var reqType HandlerMessageType
	switch r.RequestURI {
	case APIBasePath + "/base":
		reqType = TypeGetBase
	case APIBasePath + "/shoulder":
		reqType = TypeGetShoulder
	case APIBasePath + "/elbow":
		reqType = TypeGetElbow
	case APIBasePath + "/wrist/angle":
		reqType = TypeGetWristAngle
	case APIBasePath + "/wrist/rotation":
		reqType = TypeGetWristRotation
	case APIBasePath + "/gripper":
		reqType = TypeGetGripper
	case APIBasePath + "/posture":
		getPosture(w, r)
		return
	default:
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}

	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  reqType,
		Value: []interface{}{},
	}

	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	if !ok {
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}

	// respond with the result
	var name string
	switch msg.Type {
	case TypeCurrentBase:
		name = "base"
	case TypeCurrentShoulder:
		name = "shoulder"
	case TypeCurrentElbow:
		name = "elbow"
	case TypeCurrentWristAngle:
		name = "wrist/angle"
	case TypeCurrentWristRotation:
		name = "wrist/rotation"
	case TypeCurrentGripper:
		name = "gripper"
	default: // something went wrong
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	val, ok := msg.Value[0].(uint16)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
	}
	log.Printf("%s: %v", name, val)
	jointInfo := &JointInfo{name, val}
	js, err := json.Marshal(jointInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}

// putState sets the state for a joint
func putState(w http.ResponseWriter, r *http.Request) {
	var reqType HandlerMessageType
	switch r.RequestURI {
	case APIBasePath + "/base":
		reqType = TypePutBase
	case APIBasePath + "/shoulder":
		reqType = TypePutShoulder
	case APIBasePath + "/elbow":
		reqType = TypePutElbow
	case APIBasePath + "/wrist/angle":
		reqType = TypePutWristAngle
	case APIBasePath + "/wrist/rotation":
		reqType = TypePutWristRotation
	case APIBasePath + "/gripper":
		reqType = TypePutGripper
	case APIBasePath + "/posture":
		putPosture(w, r)
		return
	case APIBasePath + "/reset":
		putReset(w, r)
		return
	default:
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
	}

	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var robotCommand RobotCommand
	err := decoder.Decode(&robotCommand)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}

	// extract token from the X-API-Key header
	if token := r.Header.Get("X-API-Key"); token != "" {
		robotCommand.Token = token
	} else {
		w.WriteHeader(http.StatusBadRequest) // 401
		return
	}

	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  reqType,
		Value: []interface{}{robotCommand},
	}

	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	if !ok {
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}

	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Printf("robotCommand.Value: %v", robotCommand.Value)
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidCommand: // the invalid value provided
		log.Printf("InvalidCommand: %v", robotCommand.Value)
		w.WriteHeader(http.StatusBadRequest) // 400
	case TypeInvalidToken: // the invalid token provided
		log.Printf("InvalidToken: %v", robotCommand.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	case TypeUserNotFound: // the user not found
		log.Println("UserNotFound")
		w.WriteHeader(http.StatusBadRequest) // 400
	default: // something went wrong
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

// putPosture sets all the joints at once
func putPosture(w http.ResponseWriter, r *http.Request) {
	// parse the request body
	decoder := json.NewDecoder(r.Body)
	var posCom PostureCommand
	err := decoder.Decode(&posCom)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}

	// extract token from the X-API-Key header
	if token := r.Header.Get("X-API-Key"); token != "" {
		posCom.Token = token
	} else {
		w.WriteHeader(http.StatusBadRequest) // 401
		return
	}

	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutPosture,
		Value: []interface{}{posCom},
	}

	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	if !ok {
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}

	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Println("Posture")
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidToken: // the invalid token provided
		log.Printf("InvalidToken: %v", posCom.Token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	case TypeUserNotFound: // the user not found
		log.Println("UserNotFound")
		w.WriteHeader(http.StatusBadRequest) // 400
	default: // something went wrong
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

// putReset resets the states
func putReset(w http.ResponseWriter, r *http.Request) {
	// extract token from the X-API-Key header
	token := r.Header.Get("X-API-Key")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest) // 401
		return
	}

	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypePutReset,
		Value: []interface{}{token},
	}

	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	if !ok {
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}

	// respond with the result
	switch msg.Type {
	case TypeActionPerformed: // the requested action is performed
		log.Println("Posture")
		w.WriteHeader(http.StatusAccepted) // 202
	case TypeInvalidToken: // the invalid token provided
		log.Printf("InvalidToken: %v", token)
		w.WriteHeader(http.StatusUnauthorized) // 401
	case TypeUserNotFound: // the user not found
		log.Println("UserNotFound")
		w.WriteHeader(http.StatusBadRequest) // 400
	default: // something went wrong
		log.Printf("%#v", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError) // 500
	}
}

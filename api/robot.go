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

// GetPosture gets the current posture
func GetPosture(w http.ResponseWriter, r *http.Request) {
	// bypass the request to HandlerChannel
	HandlerChannel <- HandlerMessage{
		Type:  TypeGetPosture,
		Value: []interface{}{},
	}
	// receive a message from the other end of HandlerChannel
	msg, ok := <-HandlerChannel
	// check the channel status
	if !ok {
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

// GetState gets the current value for each joint
func GetState(w http.ResponseWriter, r *http.Request) {
	var reqType HandlerMessageType

	switch r.RequestURI {
	case "/leubot/base":
		reqType = TypeGetBase
	case "/leubot/shoulder":
		reqType = TypeGetShoulder
	case "/leubot/elbow":
		reqType = TypeGetElbow
	case "/leubot/wrist/angle":
		reqType = TypeGetWristAngle
	case "/leubot/wrist/rotation":
		reqType = TypeGetWristRotation
	case "/leubot/gripper":
		reqType = TypeGetGripper
	default:
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
	// check the channel status
	if !ok {
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
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	val, ok := msg.Value[0].(uint16)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
	}
	log.Printf("[HandlerChannel] %s: %v", name, val)
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

// PutBase processes the request
func PutBase(w http.ResponseWriter, r *http.Request) {
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

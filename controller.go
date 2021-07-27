package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Interactions-HSG/leubot/api"
	"github.com/Interactions-HSG/leubot/armlink"
	"github.com/badoux/checkmail"
)

// Controller is the main thread for this API provider
type Controller struct {
	ArmLinkSerial     *armlink.ArmLinkSerial
	CurrentRobotPose  *api.RobotPose
	CurrentUser       *api.User
	HandlerChannel    chan api.HandlerMessage
	LastArmLinkPacket *armlink.ArmLinkPacket
	MasterToken       string
	UserActChannel    chan bool
	UserTimer         *time.Timer
	UserTimerFinish   chan bool
	Version           string
}

// InitRobot initialize the robot
func (controller *Controller) InitRobot() {
	// turn on the light
	switchLight(true)
	// set the robot in Joint mode and go to home
	alp := &armlink.ArmLinkPacket{}
	alp.SetExtended(armlink.ExtendedReset)
	controller.ArmLinkSerial.Send(alp.Bytes())
	// reset CurrentRobotPose
	controller.ResetPose()
	// sync with Leubot
	alp = controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
	controller.ArmLinkSerial.Send(alp.Bytes())
	log.Printf("[ArmLinkPacket] %v", alp.String())
	// post to Slack - stop
	postToSlack(fmt.Sprintf(`{"text":"<!here> User %v (%v) started using Leubot."}`, controller.CurrentUser.Name, controller.CurrentUser.Email))
	// start the timer
	if *userTimeout != 0 {
		controller.UserTimer.Reset(time.Second * time.Duration(*userTimeout))
		log.Printf("[UserTimer] Started for %v", controller.CurrentUser.ToUserInfo().Name)
		go func() {
			for {
				select {
				case <-controller.UserActChannel: // Upon any activity, reset the timer
					log.Println("[UserTimer] Activity detected, resetting the timer")
					controller.UserTimer.Reset(time.Second * time.Duration(*userTimeout))
				case <-controller.UserTimer.C: // Inactive, logout
					log.Printf("[UserTimer] Timeout, deleting the user %v", controller.CurrentUser.Name)
					// reset CurrentRobotPose
					controller.ResetPose()
					// set the robot in sleep mode
					alp := armlink.ArmLinkPacket{}
					alp.SetExtended(armlink.ExtendedSleep)
					controller.ArmLinkSerial.Send(alp.Bytes())
					// turn off the light
					switchLight(false)
					// post to Slack
					postToSlack(fmt.Sprintf(`{"text":"<!here> User %v (%v) was inactive for %v seconds, releasing Leubot."}`, controller.CurrentUser.Name, controller.CurrentUser.Email, *userTimeout))
					// delete the current user; assign an empty User
					controller.CurrentUser = &api.User{}
					// exiting timer channel listener
					return
				case <-controller.UserTimerFinish:
					log.Println("[UserTimer] User deleted, terminating the timer")
					return
				}
			}
		}()
	} // End if *userTimeout != 0

}

// Validate checks if the given token is valid
func (controller *Controller) Validate(token string) bool {
	if token == controller.CurrentUser.Token {
		return true
	} else if token == controller.MasterToken {
		if *controller.CurrentUser == (api.User{}) {
			// register a super user
			controller.CurrentUser = api.NewUser(&api.UserInfo{
				Name:  "Super User",
				Email: "super-user@interactions.ics.unisg.ch",
			})

		}
		return true
	}
	return false
}

// ResetPose resets the RobotPose to its home position
func (controller *Controller) ResetPose() {
	controller.CurrentRobotPose = &api.RobotPose{
		Base:          512,
		Shoulder:      400,
		Elbow:         400,
		WristAngle:    580,
		WristRotation: 512,
		Gripper:       128,
	}
}

// Shutdown processes the graceful termination of the program
func (controller *Controller) Shutdown() {
	// set the robot in sleep mode
	alp := armlink.ArmLinkPacket{}
	alp.SetExtended(armlink.ExtendedSleep)
	controller.ArmLinkSerial.Send(alp.Bytes())
	// turn off the light
	switchLight(false)
}

// NewController creates a new instance of Controller
func NewController(als *armlink.ArmLinkSerial, mt string, ver string) *Controller {
	hmc := make(chan api.HandlerMessage)
	controller := Controller{
		ArmLinkSerial:     als,
		CurrentRobotPose:  &api.RobotPose{},
		CurrentUser:       &api.User{},
		HandlerChannel:    hmc,
		LastArmLinkPacket: &armlink.ArmLinkPacket{},
		MasterToken:       mt,
		UserActChannel:    make(chan bool),
		UserTimer:         time.NewTimer(time.Second * 10),
		UserTimerFinish:   make(chan bool),
		Version:           ver,
	}
	controller.ResetPose()
	controller.UserTimer.Stop()

	// init
	// set the robot in sleep mode
	alp := armlink.ArmLinkPacket{}
	alp.SetExtended(armlink.ExtendedSleep)
	controller.ArmLinkSerial.Send(alp.Bytes())
	// turn off the light
	switchLight(false)

	go func() {
		for {
			msg, ok := <-hmc
			if !ok {
				break
			}

			log.Printf("[controller.go] %v", controller.CurrentRobotPose.String())
			switch msg.Type {
			case api.TypeAddUser:
				userInfo, ok := msg.Value[0].(api.UserInfo)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the email is valid
				if err := checkmail.ValidateFormat(userInfo.Email); err != nil {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidUserInfo,
					}
					break
				}
				// check if there's no user in the system
				if controller.CurrentUser.ToUserInfo() != (api.UserInfo{}) && userInfo.Email != controller.CurrentUser.Email {
					hmc <- api.HandlerMessage{
						Type: api.TypeUserExisted,
					}
					break
				}
				// reissue the token for the existing user an return
				if userInfo.Email == controller.CurrentUser.Email {
					controller.CurrentUser = api.NewUser(&userInfo)
					log.Printf("[User] Token reissued for %v", userInfo.Name)
					controller.UserTimer.Reset(time.Second * time.Duration(*userTimeout))
					log.Println("[UserTimer] Timer resetted")
					// skip the rest and return the response with the new token
					hmc <- api.HandlerMessage{
						Type:  api.TypeUserAdded,
						Value: []interface{}{*controller.CurrentUser},
					}
					break
				}
				// register the user to the system with the new token
				controller.CurrentUser = api.NewUser(&userInfo)
				// initialize the robot
				controller.InitRobot()
				// respond to the hmc
				hmc <- api.HandlerMessage{
					Type:  api.TypeUserAdded,
					Value: []interface{}{*controller.CurrentUser},
				}
			case api.TypeGetUser:
				hmc <- api.HandlerMessage{
					Type:  api.TypeCurrentUser,
					Value: []interface{}{controller.CurrentUser.ToUserInfo()},
				}
			case api.TypeDeleteUser:
				// receive the token
				token, ok := msg.Value[0].(string)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if !controller.Validate(token) {
					hmc <- api.HandlerMessage{
						Type: api.TypeUserNotFound,
					}
					break
				}
				// stop the timer
				if *userTimeout != 0 {
					controller.UserTimer.Stop()
					controller.UserTimerFinish <- true
				}
				// reset CurrentRobotPose
				controller.ResetPose()
				// set the robot in sleep mode
				alp := armlink.ArmLinkPacket{}
				alp.SetExtended(armlink.ExtendedSleep)
				controller.ArmLinkSerial.Send(alp.Bytes())
				// turn off the light
				switchLight(false)
				// post to Slack - start
				postToSlack(fmt.Sprintf(`{"text":"<!here> User %v (%v) started using Leubot."}`, controller.CurrentUser.Name, controller.CurrentUser.Email))
				// delete the current user; assign an empty User
				controller.CurrentUser = &api.User{}

				hmc <- api.HandlerMessage{
					Type: api.TypeUserDeleted,
				}
			case api.TypeGetBase:
				hmc <- api.HandlerMessage{
					Type:  api.TypeCurrentBase,
					Value: []interface{}{controller.CurrentRobotPose.Base},
				}
			case api.TypeGetShoulder:
				hmc <- api.HandlerMessage{
					Type:  api.TypeCurrentShoulder,
					Value: []interface{}{controller.CurrentRobotPose.Shoulder},
				}
			case api.TypeGetElbow:
				hmc <- api.HandlerMessage{
					Type:  api.TypeCurrentElbow,
					Value: []interface{}{controller.CurrentRobotPose.Elbow},
				}
			case api.TypeGetWristAngle:
				hmc <- api.HandlerMessage{
					Type:  api.TypeCurrentWristAngle,
					Value: []interface{}{controller.CurrentRobotPose.WristAngle},
				}
			case api.TypeGetWristRotation:
				hmc <- api.HandlerMessage{
					Type:  api.TypeCurrentWristRotation,
					Value: []interface{}{controller.CurrentRobotPose.WristRotation},
				}
			case api.TypeGetGripper:
				hmc <- api.HandlerMessage{
					Type:  api.TypeCurrentGripper,
					Value: []interface{}{controller.CurrentRobotPose.Gripper},
				}
			case api.TypeGetPosture:
				hmc <- api.HandlerMessage{
					Type:  api.TypeCurrentPosture,
					Value: []interface{}{*controller.CurrentRobotPose},
				}
			case api.TypePutBase:
				// check if there's a user
				if controller.CurrentUser.ToUserInfo() == (api.UserInfo{}) {
					// don't allow if not activated
					hmc <- api.HandlerMessage{
						Type: api.TypeUserNotFound,
					}
					break
				}
				// receive the robotCommand
				robotCommand, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if !controller.Validate(robotCommand.Token) {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidToken,
					}
					break
				}
				// check the value is valid
				if robotCommand.Value < 0 || 1023 < robotCommand.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}
				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}
				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.Base = robotCommand.Value
				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutShoulder:
				// check if there's a user
				if controller.CurrentUser.ToUserInfo() == (api.UserInfo{}) {
					// don't allow if not activated
					hmc <- api.HandlerMessage{
						Type: api.TypeUserNotFound,
					}
					break
				}
				// receive the robotCommand
				robotCommand, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if !controller.Validate(robotCommand.Token) {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidToken,
					}
					break
				}
				// check the value is valid
				if robotCommand.Value < 205 || 810 < robotCommand.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}
				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}
				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.Shoulder = robotCommand.Value
				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutElbow:
				// check if there's a user
				if controller.CurrentUser.ToUserInfo() == (api.UserInfo{}) {
					// don't allow if not activated
					hmc <- api.HandlerMessage{
						Type: api.TypeUserNotFound,
					}
					break
				}
				// receive the robotCommand
				robotCommand, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if !controller.Validate(robotCommand.Token) {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidToken,
					}
					break
				}
				// check the value is valid
				if robotCommand.Value < 210 || 900 < robotCommand.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}
				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}
				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.Elbow = robotCommand.Value
				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutWristAngle:
				// check if there's a user
				if controller.CurrentUser.ToUserInfo() == (api.UserInfo{}) {
					// don't allow if not activated
					hmc <- api.HandlerMessage{
						Type: api.TypeUserNotFound,
					}
					break
				}
				// receive the robotCommand
				robotCommand, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if !controller.Validate(robotCommand.Token) {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidToken,
					}
					break
				}
				// check the value is valid
				if robotCommand.Value < 200 || 830 < robotCommand.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}
				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}
				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.WristAngle = robotCommand.Value
				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutWristRotation:
				// check if there's a user
				if controller.CurrentUser.ToUserInfo() == (api.UserInfo{}) {
					// don't allow if not activated
					hmc <- api.HandlerMessage{
						Type: api.TypeUserNotFound,
					}
					break
				}
				// receive the robotCommand
				robotCommand, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if !controller.Validate(robotCommand.Token) {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidToken,
					}
					break
				}
				// check the value is valid
				if robotCommand.Value < 0 || 1023 < robotCommand.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}
				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}
				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.WristRotation = robotCommand.Value
				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutGripper:
				// check if there's a user
				if controller.CurrentUser.ToUserInfo() == (api.UserInfo{}) {
					// don't allow if not activated
					hmc <- api.HandlerMessage{
						Type: api.TypeUserNotFound,
					}
					break
				}
				// receive the robotCommand
				robotCommand, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if !controller.Validate(robotCommand.Token) {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidToken,
					}
					break
				}
				// check the value is valid
				if robotCommand.Value < 0 || 512 < robotCommand.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}
				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}
				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.Gripper = robotCommand.Value
				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutPosture:
				// check if there's a user
				if controller.CurrentUser.ToUserInfo() == (api.UserInfo{}) {
					// don't allow if not activated
					hmc <- api.HandlerMessage{
						Type: api.TypeUserNotFound,
					}
					break
				}
				// receive the posCom
				posCom, ok := msg.Value[0].(api.PostureCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if !controller.Validate(posCom.Token) {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidToken,
					}
					break
				}
				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}
				// check the value is valid
				log.Printf("[Posture] %v", posCom)
				if posCom.Base < 0 || 1023 < posCom.Base ||
					posCom.Shoulder < 205 || 810 < posCom.Shoulder ||
					posCom.Elbow < 210 || 900 < posCom.Elbow ||
					posCom.WristAngle < 200 || 830 < posCom.WristAngle ||
					posCom.WristRotation < 0 || 1023 < posCom.WristRotation ||
					posCom.Gripper < 0 || 512 < posCom.Gripper ||
					posCom.Delta < 0 || 254 < posCom.Delta {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}
				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.Base = posCom.Base
				controller.CurrentRobotPose.Shoulder = posCom.Shoulder
				controller.CurrentRobotPose.Elbow = posCom.Elbow
				controller.CurrentRobotPose.WristAngle = posCom.WristAngle
				controller.CurrentRobotPose.WristRotation = posCom.WristRotation
				controller.CurrentRobotPose.Gripper = posCom.Gripper
				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(posCom.Delta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutReset:
				// receive the robotCommand
				token, ok := msg.Value[0].(string)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if !controller.Validate(token) {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidToken,
					}
					break
				}
				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}
				// perform the reset
				alp := &armlink.ArmLinkPacket{}
				alp.SetExtended(armlink.ExtendedReset)
				controller.ArmLinkSerial.Send(alp.Bytes())
				// reset CurrentRobotPose
				controller.ResetPose()
				// sync with Leubot
				alp = controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			}
		}
		log.Fatalln("HandlerChannel closed, dying...")
	}()

	return &controller
}

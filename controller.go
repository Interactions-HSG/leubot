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
	CurrentRobotState RobotState
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
					controller.SleepRobot()
					// post to Slack
					postToSlack(fmt.Sprintf(`{"text":"<!here> User %v (%v) was inactive for %v seconds, releasing Leubot."}`, controller.CurrentUser.Name, controller.CurrentUser.Email, *userTimeout))

					// delete the current user; assign an empty User
					controller.CurrentUser = &api.User{}
					return
				case <-controller.UserTimerFinish:
					log.Println("[UserTimer] User deleted, terminating the timer")
					return
				}
			}
		}()
	} // End if *userTimeout != 0
	controller.CurrentRobotState = Ready
}

// SleepRobot sleeps the robot
func (controller *Controller) SleepRobot() {
	alp := armlink.ArmLinkPacket{}
	alp.SetExtended(armlink.ExtendedSleep)
	controller.ArmLinkSerial.Send(alp.Bytes())
	// turn off the light
	switchLight(false)
	// zero out the CurrentRobotPose
	controller.CurrentRobotPose = &api.RobotPose{
		Base:          0,
		Shoulder:      0,
		Elbow:         0,
		WristAngle:    0,
		WristRotation: 0,
		Gripper:       0,
	}
	// enter sleeping state
	controller.CurrentRobotState = Sleeping
}

// Validate checks if the given token is valid, if the token is master token
// and there's no user then create a super user
func (controller *Controller) Validate(token string) api.HandlerMessageType {
	log.Printf("Validate the token: %v", token)
	if token == controller.MasterToken {
		if *controller.CurrentUser == (api.User{}) {
			// register a super user
			log.Println("Create a super user")
			controller.CurrentUser = &api.User{
				Name:  "Super User",
				Email: "root@interactions.ics.unisg.ch",
				Token: token,
			}
			controller.UserTimer.Reset(time.Second * time.Duration(*userTimeout))

			// initialize the robot
			controller.InitRobot()
			return api.TypeUserAdded
		}
		return api.TypeUserExisted
	} else if controller.CurrentUser.ToUserInfo() == (api.UserInfo{}) {
		// no user exists
		return api.TypeUserNotFound
	} else if token == controller.CurrentUser.Token {
		return api.TypeUserExisted
	}
	return api.TypeInvalidToken
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
		CurrentRobotState: Offline,
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

	// set the robot in sleep mode
	controller.SleepRobot()

	go func() {
		for {
			msg, ok := <-hmc
			if !ok {
				break
			}

			log.Printf("%v", controller.CurrentRobotPose.String())
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
					log.Printf("Token reissued for %v", userInfo.Name)
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

				// feedback
				hmc <- api.HandlerMessage{
					Type:  api.TypeUserAdded,
					Value: []interface{}{*controller.CurrentUser},
				}
			case api.TypeGetUser:
				// feedback
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
				userAuth := controller.Validate(token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
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
				controller.SleepRobot()

				// post to Slack - start
				postToSlack(fmt.Sprintf(`{"text":"<!here> User %v (%v) started using Leubot."}`, controller.CurrentUser.Name, controller.CurrentUser.Email))

				// delete the current user; assign an empty User
				controller.CurrentUser = &api.User{}

				// feedback
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
				// receive the roboCom
				roboCom, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}

				// check if the token is valid
				userAuth := controller.Validate(roboCom.Token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
					}
					break
				}

				// check the value is valid
				if roboCom.Value < 0 || 1023 < roboCom.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}

				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}

				// wake up if sleeping
				if controller.CurrentRobotState == Sleeping {
					log.Println("Leubot is sleeping, waking up")
					controller.InitRobot()
				}

				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.Base = roboCom.Value

				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				// feedback
				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutShoulder:
				// receive the roboCom
				roboCom, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}

				// check if the token is valid
				userAuth := controller.Validate(roboCom.Token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
					}
					break
				}

				// check the value is valid
				if roboCom.Value < 205 || 810 < roboCom.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}

				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}

				// wake up if sleeping
				if controller.CurrentRobotState == Sleeping {
					log.Println("Leubot is sleeping, waking up")
					controller.InitRobot()
				}

				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.Shoulder = roboCom.Value

				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				// feedback
				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutElbow:
				// receive the roboCom
				roboCom, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}

				// check if the token is valid
				userAuth := controller.Validate(roboCom.Token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
					}
					break
				}

				// check the value is valid
				if roboCom.Value < 210 || 900 < roboCom.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}

				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}

				// wake up if sleeping
				if controller.CurrentRobotState == Sleeping {
					log.Println("Leubot is sleeping, waking up")
					controller.InitRobot()
				}

				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.Elbow = roboCom.Value

				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				// feedback
				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutWristAngle:
				// receive the roboCom
				roboCom, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}

				// check if the token is valid
				userAuth := controller.Validate(roboCom.Token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
					}
					break
				}

				// check the value is valid
				if roboCom.Value < 200 || 830 < roboCom.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}

				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}

				// wake up if sleeping
				if controller.CurrentRobotState == Sleeping {
					log.Println("Leubot is sleeping, waking up")
					controller.InitRobot()
				}

				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.WristAngle = roboCom.Value

				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				// feedback
				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutWristRotation:
				// receive the roboCom
				roboCom, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}

				// check if the token is valid
				userAuth := controller.Validate(roboCom.Token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
					}
					break
				}

				// check the value is valid
				if roboCom.Value < 0 || 1023 < roboCom.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}

				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}

				// wake up if sleeping
				if controller.CurrentRobotState == Sleeping {
					log.Println("Leubot is sleeping, waking up")
					controller.InitRobot()
				}

				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.WristRotation = roboCom.Value

				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				// feedback
				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutGripper:
				// receive the roboCom
				roboCom, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}

				// check if the token is valid
				userAuth := controller.Validate(roboCom.Token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
					}
					break
				}

				// check the value is valid
				if roboCom.Value < 0 || 512 < roboCom.Value {
					hmc <- api.HandlerMessage{
						Type: api.TypeInvalidCommand,
					}
					break
				}

				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}

				// wake up if sleeping
				if controller.CurrentRobotState == Sleeping {
					log.Println("Leubot is sleeping, waking up")
					controller.InitRobot()
				}

				// set the value to CurrentRobotPose
				controller.CurrentRobotPose.Gripper = roboCom.Value

				// perform the move
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(*defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())

				// feedback
				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutPosture:
				// receive the posCom
				posCom, ok := msg.Value[0].(api.PostureCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}

				// check if the token is valid
				userAuth := controller.Validate(posCom.Token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
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

				// wake up if sleeping
				if controller.CurrentRobotState == Sleeping {
					log.Println("Leubot is sleeping, waking up")
					controller.InitRobot()
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

				// feedback
				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutReset:
				// receive the token
				token, ok := msg.Value[0].(string)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}

				// check if the token is valid
				userAuth := controller.Validate(token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
					}
					break
				}

				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}

				// wake up if sleeping
				if controller.CurrentRobotState == Sleeping {
					log.Println("Leubot is sleeping, waking up")
					controller.InitRobot()
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

				// feedback
				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			case api.TypePutSleep:
				// receive the token
				token, ok := msg.Value[0].(string)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}

				// check if the token is valid
				userAuth := controller.Validate(token)
				if userAuth != api.TypeUserExisted && userAuth != api.TypeUserAdded {
					// feedback
					hmc <- api.HandlerMessage{
						Type: userAuth,
					}
					break
				}

				// ack the timer
				if *userTimeout != 0 {
					controller.UserActChannel <- true
				}

				// sleep if it's Ready
				if controller.CurrentRobotState == Ready {
					// reset CurrentRobotPose
					controller.ResetPose()

					// set the robot in sleep mode
					controller.SleepRobot()
				}

				// feedback
				hmc <- api.HandlerMessage{
					Type: api.TypeActionPerformed,
				}
			}
		}
		log.Fatalln("HandlerChannel closed, dying...")
	}()

	return &controller
}

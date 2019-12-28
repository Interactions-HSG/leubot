/*
 * Leubot
 *
 * This program provides a simple API for
 * PhantomX AX-12 Reactor Robot Arm with ArmLink Serial interface
 *
 * Contact: iori.mizutani@unisg.ch
 */

package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/Interactions-HSG/leubot/api"
	"github.com/Interactions-HSG/leubot/armlink"
	"github.com/badoux/checkmail"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Environmental variables
var (
	// Current Version
	version = "1.1.1"

	// app
	app = kingpin.
		New("leubot", "Provide a Web API for the PhantomX AX-12 Reactor Robot Arm.")

	// delta for the leubot
	defaultDelta = uint8(128)

	// flags
	mastertoken = app.
			Flag("mastertoken", "The master token for debug.").
			Default("sometoken").
			String()

	miioenabled = app.
			Flag("miioenabled", "Enable Xiaomi yeelight device.").
			Default("false").
			Bool()

	miiocli = app.
		Flag("miiocli", "The path to miio cli.").
		Default("/opt/bin/miiocli").
		String()

	miiotoken = app.
			Flag("miiotoken", "The token for Xiaomi yeelight device.").
			Default("0000000000000000000000000000").
			String()

	miioip = app.
		Flag("miioip", "The IP address for Xiaomi yeelight device.").
		Default("192.168.1.2").
		String()

	slackappenabled = app.
			Flag("slackappenabled", "Enable Slack app for user previleges.").
			Default("false").
			Bool()
	slackwebhookurl = app.
			Flag("slackwebhookurl", "The webhook url for posting the json payloads.").
			Default("https://hooks.slack.com/services/...").
			String()

	userTimeout = app.
			Flag("userTimeout", "The timeout duration for users in seconds.").
			Default("900").
			Int()
)

// Controller is the main thread for this API provider
type Controller struct {
	ArmLinkSerial     *armlink.ArmLinkSerial
	CurrentRobotPose  *api.RobotPose
	CurrentUser       *api.User
	HandlerChannel    chan api.HandlerMessage
	LastArmLinkPacket *armlink.ArmLinkPacket
	UserActChannel    chan bool
	UserTimer         *time.Timer
	UserTimerFinish   chan bool
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
func NewController(als *armlink.ArmLinkSerial) *Controller {
	hmc := make(chan api.HandlerMessage)
	controller := Controller{
		ArmLinkSerial:     als,
		CurrentRobotPose:  &api.RobotPose{},
		CurrentUser:       &api.User{},
		HandlerChannel:    hmc,
		LastArmLinkPacket: &armlink.ArmLinkPacket{},
		UserActChannel:    make(chan bool),
		UserTimer:         time.NewTimer(time.Second * 10),
		UserTimerFinish:   make(chan bool),
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

			log.Printf("[CurrentRobotPose] %v", controller.CurrentRobotPose.String())

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
				// turn on the light
				if *miioenabled {
					cmd := exec.Command(*miiocli, "yeelight", "--ip", *miioip, "--token", *miiotoken, "on")
					cmd.Run()
				}
				// set the robot in Joint mode and go to home
				alp := &armlink.ArmLinkPacket{}
				alp.SetExtended(armlink.ExtendedReset)
				controller.ArmLinkSerial.Send(alp.Bytes())
				// reset CurrentRobotPose
				controller.ResetPose()
				// sync with Leubot
				alp = controller.CurrentRobotPose.BuildArmLinkPacket(defaultDelta)
				controller.ArmLinkSerial.Send(alp.Bytes())
				log.Printf("[ArmLinkPacket] %v", alp.String())
				// post to Slack - stop
				postToSlack(fmt.Sprintf(`{"text":"<!here> User %v (%v) stopped using Leubot."}`, controller.CurrentUser.Name, controller.CurrentUser.Email))
				// start the timer
				if *userTimeout != 0 {
					controller.UserTimer.Reset(time.Second * time.Duration(*userTimeout))
					log.Printf("[UserTimer] Started for %v", userInfo.Name)
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
				if token != controller.CurrentUser.Token && token != *mastertoken {
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
				if robotCommand.Token != controller.CurrentUser.Token && robotCommand.Token != *mastertoken {
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
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(defaultDelta)
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
				if robotCommand.Token != controller.CurrentUser.Token && robotCommand.Token != *mastertoken {
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
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(defaultDelta)
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
				if robotCommand.Token != controller.CurrentUser.Token && robotCommand.Token != *mastertoken {
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
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(defaultDelta)
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
				if robotCommand.Token != controller.CurrentUser.Token && robotCommand.Token != *mastertoken {
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
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(defaultDelta)
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
				if robotCommand.Token != controller.CurrentUser.Token && robotCommand.Token != *mastertoken {
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
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(defaultDelta)
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
				if robotCommand.Token != controller.CurrentUser.Token && robotCommand.Token != *mastertoken {
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
				alp := controller.CurrentRobotPose.BuildArmLinkPacket(defaultDelta)
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
				if posCom.Token != controller.CurrentUser.Token && posCom.Token != *mastertoken {
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
				robotCommand, ok := msg.Value[0].(api.RobotCommand)
				if !ok {
					hmc <- api.HandlerMessage{
						Type: api.TypeSomethingWentWrong,
					}
					break
				}
				// check if the token is valid
				if robotCommand.Token != controller.CurrentUser.Token && robotCommand.Token != *mastertoken {
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
				alp = controller.CurrentRobotPose.BuildArmLinkPacket(defaultDelta)
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

// postToSlack posts the status to Slack if slackappenabled
func postToSlack(msg string) {
	if *slackappenabled {
		var jsonStr = []byte(msg)
		req, err := http.NewRequest("POST", *slackwebhookurl, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")
		r, err := (&http.Client{}).Do(req)
		if err != nil {
			panic(err)
		}
		r.Body.Close()
	}
}

// switchLight turns on/off the light if miioenabled
func switchLight(on bool) {
	if *miioenabled {
		stateOnOff := "on"
		if !on {
			stateOnOff = "off"
		}
		cmd := exec.Command(*miiocli, "yeelight", "--ip", *miioip, "--token", *miiotoken, stateOnOff)
		cmd.Run()
	}
}

func main() {
	app.Version(version)
	parse := kingpin.MustParse(app.Parse(os.Args[1:]))
	_ = parse

	// initialize ArmLink serial interface to control the robot
	als := armlink.NewArmLinkSerial()
	defer als.Close()

	// create the controller with the serial
	controller := NewController(als)
	defer controller.Shutdown()

	log.Printf("Server started")
	router := api.NewRouter(controller.HandlerChannel)
	log.Fatal(http.ListenAndServe(":6789", router))
}

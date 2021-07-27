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
	//"runtime/debug"

	"github.com/Interactions-HSG/leubot/api"
	"github.com/Interactions-HSG/leubot/armlink"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Environmental variables
var (
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

	serverIP = app.
			Flag("ip", "The IP address of the Leubot server.").
			Default("172.0.0.1").
			String()

	serverPort = app.
			Flag("port", "The serving port of the Leubot server.").
			Default("6789").
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
	parse := kingpin.MustParse(app.Parse(os.Args[1:]))
	_ = parse

	//bi, _ := debug.ReadBuildInfo()
	//app.Version(bi.Main.Version)
	//log.Printf("Server started: %v", bi.Main.Version)

	// initialize ArmLink serial interface to control the robot
	als := armlink.NewArmLinkSerial()
	defer als.Close()

	// create the controller with the serial
	controller := NewController(als)
	defer controller.Shutdown()

	router := api.NewRouter(controller.HandlerChannel)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%v:%v", *serverIP, *serverPort), router))
}

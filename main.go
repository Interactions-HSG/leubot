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
	"runtime/debug"

	"github.com/Interactions-HSG/leubot/api"
	"github.com/Interactions-HSG/leubot/armlink"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Environmental variables
var (
	defaultDelta = uint8(128)

	app             = kingpin.New("leubot", "Provide a Web API for the PhantomX AX-12 Reactor Robot Arm.")
	apiHost         = app.Flag("apiHost", "The hostname for the API.").Default("api.interactions.ics.unisg.ch").String()
	apiPath         = app.Flag("apiPath", "The name for the path.").Default("leubot").String()
	apiProto        = app.Flag("apiProto", "The protocol for the API.").Default("https://").String()
	masterToken     = app.Flag("masterToken", "The master token for debug.").Default("sometoken").String()
	miioEnabled     = app.Flag("miioEnabled", "Enable Xiaomi yeelight device.").Default("false").Bool()
	miiocliPath     = app.Flag("miiocliPath", "The path to miio cli.").Default("/opt/bin/miiocli").String()
	miioToken       = app.Flag("miioToken", "The token for Xiaomi yeelight device.").Default("0000000000000000000000000000").String()
	miioIP          = app.Flag("miioIP", "The IP address for Xiaomi yeelight device.").Default("192.168.1.2").String()
	serverIP        = app.Flag("ip", "The IP address of the Leubot server.").Default("172.0.0.1").String()
	serverPort      = app.Flag("port", "The serving port of the Leubot server.").Default("6789").String()
	slackAppEnabled = app.Flag("slackAppEnabled", "Enable Slack app for user previleges.").Default("false").Bool()
	slackWebHookURL = app.Flag("slackWebHookURL", "The webhook url for posting the json payloads.").Default("https://hooks.slack.com/services/...").String()
	userTimeout     = app.Flag("userTimeout", "The timeout duration for users in seconds.").Default("900").Int()
)

// postToSlack posts the status to Slack if slackAppEnabled
func postToSlack(msg string) {
	if *slackAppEnabled {
		var jsonStr = []byte(msg)
		req, err := http.NewRequest("POST", *slackWebHookURL, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")
		r, err := (&http.Client{}).Do(req)
		if err != nil {
			panic(err)
		}
		r.Body.Close()
	}
}

// switchLight turns on/off the light if miioEnabled
func switchLight(on bool) {
	if *miioEnabled {
		stateOnOff := "on"
		if !on {
			stateOnOff = "off"
		}
		cmd := exec.Command(*miiocliPath, "yeelight", "--ip", *miioIP, "--token", *miioToken, stateOnOff)
		cmd.Run()
	}
}

func main() {
	bi, ok := debug.ReadBuildInfo()
	var version string
	if ok {
		version = bi.Main.Version
	} else {
		version = "v1.2"
	}
	app.Version(version)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case "version":
		fmt.Println(version)
		os.Exit(0)
	}
	log.Printf("Leubot (%v) started", version)

	// initialize ArmLink serial interface to control the robot
	als := armlink.NewArmLinkSerial()
	defer als.Close()

	// create the controller with the serial
	controller := NewController(als, *masterToken, version)
	defer controller.Shutdown()

	router := api.NewRouter(*apiHost, *apiPath, *apiProto, controller.HandlerChannel, version)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%v:%v", *serverIP, *serverPort), router))
}

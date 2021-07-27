package main

// RobotState holds the state of the robot
type RobotState int

const (
	// Offline - the robot is not yet initialized
	Offline RobotState = iota
	// Ready - the robot is initialized and ready to perform actions
	Ready
	// Busy - the robot is performing an action
	Busy
	// Sleeping - the robot is sleeping
	Sleeping
)

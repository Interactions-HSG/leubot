package api

// HandlerMessageType is the type for HandlerMessage
type HandlerMessageType int

const (
	// TypeAddUser is to add a user
	TypeAddUser HandlerMessageType = iota
	// TypeUserAdded says the user is added
	TypeUserAdded
	// TypeUserExisted says a user already exists
	TypeUserExisted
	// TypeInvalidUserInfo says something wrong about the user to be added
	TypeInvalidUserInfo
	// TypeDeleteUser is to delete a user
	TypeDeleteUser
	// TypeUserDeleted says the user is deleted
	TypeUserDeleted
	// TypeUserNotFound says no such user exists
	TypeUserNotFound
	// TypeGetUser is to get the current user
	TypeGetUser
	// TypeCurrentUser has the current user info
	TypeCurrentUser
	// TypePutBase is to change Base
	TypePutBase
	// TypePutShoulder is to change Shoulder
	TypePutShoulder
	// TypePutElbow is to change Elbow
	TypePutElbow
	// TypePutWristAngle is to change WristAngle
	TypePutWristAngle
	// TypePutWristRotation is to change WristRotation
	TypePutWristRotation
	// TypePutGripper is to change Gripper
	TypePutGripper
	// TypePutPosture is to set a posture
	TypePutPosture
	// TypePutReset is to reset Leubot
	TypePutReset
	// TypeActionPerformed says the action was performed
	TypeActionPerformed
	// TypeInvalidToken says something is wrong with Token
	TypeInvalidToken
	// TypeInvalidCommand says no such command
	TypeInvalidCommand
	// TypeSomethingWentWrong says it didn't go well
	TypeSomethingWentWrong
)

// HandlerMessage contains the payload for the command messages
type HandlerMessage struct {
	Type  HandlerMessageType
	Value []interface{}
}

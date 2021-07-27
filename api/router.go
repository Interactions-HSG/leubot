package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Route is a basic pattern of the rounting
type Route struct {
	Name        string
	Methods     []string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes contain the Route
type Routes []Route

var (
	// APIBaseURL is the path to the API
	APIBaseURL string

	// APIHost is the hostname
	APIHost string

	// APIProto for API access protocol
	APIProto string

	// HandlerChannel is used to communicate between the router and other application logic
	HandlerChannel chan HandlerMessage
)

var routes = Routes{
	Route{
		"/user",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost},
		APIBaseURL + "/user",
		UserHandler,
	},
	Route{
		"/user/{token}",
		[]string{http.MethodDelete, http.MethodOptions},
		APIBaseURL + "/user/{token}",
		UserHandler,
	},
	Route{
		"GetBase",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/base",
		GetState,
	},
	Route{
		"GetShoulder",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/shoulder",
		GetState,
	},
	Route{
		"GetElbow",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/elbow",
		GetState,
	},
	Route{
		"GetWristAngle",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/wrist/angle",
		GetState,
	},
	Route{
		"GetWristRotation",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/wrist/rotation",
		GetState,
	},
	Route{
		"GetGripper",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/gripper",
		GetState,
	},
	Route{
		"GetPosture",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/posture",
		GetPosture,
	},
	Route{
		"PutBase",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/base",
		PutBase,
	},
	Route{
		"PutShoulder",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/shoulder",
		PutShoulder,
	},
	Route{
		"PutElbow",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/elbow",
		PutElbow,
	},
	Route{
		"PutWristAngle",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/wrist/angle",
		PutWristAngle,
	},
	Route{
		"PutWristRotation",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/wrist/rotation",
		PutWristRotation,
	},
	Route{
		"PutGripper",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/gripper",
		PutGripper,
	},
	Route{
		"PutPosture",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBaseURL + "/posture",
		PutPosture,
	},
	Route{
		"PutReset",
		[]string{http.MethodOptions, http.MethodPut},
		APIBaseURL + "/reset",
		PutReset,
	},
}

// Logger handles the logging in the router
func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Printf(
			"%s %s %s %s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}

// NewRouter creats a new instance of Router
func NewRouter(apiHost string, apiPath string, apiProto string, hmc chan HandlerMessage, ver string) *mux.Router {
	APIBaseURL = fmt.Sprintf("/%s/%s", apiPath, ver)
	APIHost = apiHost
	APIProto = apiProto
	HandlerChannel = hmc
	r := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)
		r.Methods(route.Methods...).Path(route.Pattern).Name(route.Name).Handler(handler)
	}
	r.Use(mux.CORSMethodMiddleware(r))

	return r
}

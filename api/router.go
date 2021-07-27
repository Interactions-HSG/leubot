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
	// APIBasePath is the path to the API
	APIBasePath string

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
		APIBasePath + "/user",
		UserHandler,
	},
	Route{
		"/user/{token}",
		[]string{http.MethodDelete, http.MethodOptions},
		APIBasePath + "/user/{token}",
		UserHandler,
	},
	Route{
		"GetBase",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/base",
		GetState,
	},
	Route{
		"GetShoulder",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/shoulder",
		GetState,
	},
	Route{
		"GetElbow",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/elbow",
		GetState,
	},
	Route{
		"GetWristAngle",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/wrist/angle",
		GetState,
	},
	Route{
		"GetWristRotation",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/wrist/rotation",
		GetState,
	},
	Route{
		"GetGripper",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/gripper",
		GetState,
	},
	Route{
		"GetPosture",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/posture",
		GetPosture,
	},
	Route{
		"PutBase",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/base",
		PutBase,
	},
	Route{
		"PutShoulder",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/shoulder",
		PutShoulder,
	},
	Route{
		"PutElbow",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/elbow",
		PutElbow,
	},
	Route{
		"PutWristAngle",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/wrist/angle",
		PutWristAngle,
	},
	Route{
		"PutWristRotation",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/wrist/rotation",
		PutWristRotation,
	},
	Route{
		"PutGripper",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/gripper",
		PutGripper,
	},
	Route{
		"PutPosture",
		[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
		APIBasePath + "/posture",
		PutPosture,
	},
	Route{
		"PutReset",
		[]string{http.MethodOptions, http.MethodPut},
		APIBasePath + "/reset",
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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// NewRouter creats a new instance of Router
func NewRouter(apiHost string, apiPath string, apiProto string, hmc chan HandlerMessage, ver string) *mux.Router {
	APIBasePath = fmt.Sprintf("/%s/%s", apiPath, ver)
	APIHost = apiHost
	APIProto = apiProto
	log.Printf("Serving at %s%s%s", APIProto, APIHost, APIBasePath)
	HandlerChannel = hmc
	r := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)
		r.Methods(route.Methods...).Path(route.Pattern).Name(route.Name).Handler(handler)
	}
	r.Use(loggingMiddleware)
	r.Use(mux.CORSMethodMiddleware(r))

	return r
}

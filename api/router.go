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

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)
}

// NewRouter creats a new instance of Router
func NewRouter(apiHost string, apiPath string, apiProto string, hmc chan HandlerMessage, ver string) *mux.Router {
	APIBasePath = fmt.Sprintf("/%s/%s", apiPath, ver)
	APIHost = apiHost
	APIProto = apiProto
	log.Printf("Serving at %s%s%s", APIProto, APIHost, APIBasePath)

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
			"/base",
			[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
			APIBasePath + "/base",
			RobotHandler,
		},
		Route{
			"/shoulder",
			[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
			APIBasePath + "/shoulder",
			RobotHandler,
		},
		Route{
			"/elbow",
			[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
			APIBasePath + "/elbow",
			RobotHandler,
		},
		Route{
			"/wrist/angle",
			[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
			APIBasePath + "/wrist/angle",
			RobotHandler,
		},
		Route{
			"/wrist/rotation",
			[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
			APIBasePath + "/wrist/rotation",
			RobotHandler,
		},
		Route{
			"/gripper",
			[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
			APIBasePath + "/gripper",
			RobotHandler,
		},
		Route{
			"/posture",
			[]string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut},
			APIBasePath + "/posture",
			RobotHandler,
		},
		Route{
			"PutReset",
			[]string{http.MethodOptions, http.MethodPut},
			APIBasePath + "/reset",
			RobotHandler,
		},
		Route{
			"PutSleep",
			[]string{http.MethodOptions, http.MethodPut},
			APIBasePath + "/sleep",
			RobotHandler,
		},
	}

	HandlerChannel = hmc
	r := mux.NewRouter().StrictSlash(true)
	// default handler
	r.Path("/").HandlerFunc(defaultHandler)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)
		r.Methods(route.Methods...).Path(route.Pattern).Name(route.Name).Handler(handler)
	}
	r.Use(mux.CORSMethodMiddleware(r))

	return r
}

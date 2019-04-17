package main

import (
	"net/http"
        "io"
	"io/ioutil"
	"github.com/go-chi/chi"
        "github.com/go-chi/chi/middleware"
        "github.com/tidwall/gjson"
        //"github.com/glennswest/libpowershell/pshell"
	"github.com/kardianos/service"
        "strings"
         //"os"
         "encoding/json"
         "fmt"
         "log"
)

var router *chi.Mux

var logSrv service.Logger
var name = "winnodeman"
var displayName = "winnodeman"
var desc = "OpenShift 4.0 Windows Node Manager"

func routers() *chi.Mux {
     router.Post("/node/install/{guid}", InstallNode)
     router.Delete("/node/uninstall/{guid}", UninstallNode)
     router.Put("/node/update/{guid}", UpdateNode)
     router.Get("/healthz",ReadyCheck)
     router.Get("/alivez", AliveCheck)
     return(router)
}

func init() { 
    router = chi.NewRouter() 
    router.Use(middleware.Recoverer)  
    router.Use(middleware.RequestID)
    router.Use(middleware.Logger)
    router.Use(middleware.Recoverer)
    router.Use(middleware.URLFormat)
}


func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}
func (p *program) run() {
	// Do work here
        EnableRestServices()
}
func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

func main() {
        s, err: = service.NewService (name, displayName, desc)
	if err! = nil {
		fmt.Printf ("% s unable to start:% s", displayName, err)
		return
	}
	logSrv = s

	if len (os.Args)> 1 {
		var err error
		verb: = os.Args [1]
		switch verb {
		case "install":
			err = s.Install ()
			if err! = nil {
				fmt.Printf ("Failed to install:% s \ n", err)
				return
			}
			fmt.Printf ("Service \"% s \ "installed. \ n", displayName)
		case "remove":
			err = s.Remove ()
			if err! = nil {
				fmt.Printf ("Failed to remove:% s \ n", err)
				return
			}
			fmt.Printf ("Service \"% s \ "removed. \ n", displayName)
		case "run":
			isService = false
			DoWork ()
		case "start":
			err = s.Start ()
			if err! = nil {
				fmt.Printf ("Failed to start:% s \ n", err)
				return
			}
			fmt.Printf ("Service \"% s \ "started. \ n", displayName)
		case "stop":
			err = s.Stop ()
			if err! = nil {
				fmt.Printf ("Failed to stop:% s \ n", err)
				return
			}
			fmt.Printf ("Service \"% s \ "stopped. \ n", displayName)
		}
		return
	}
	err = s.Run (func () error {
		// start
		go doWork ()
		return nil
	}, func () error {
		// stop
		stopWork ()
		return nil
	})
	if err! = nil {
		s.Error (err.Error ())
	}
}
}
func ReadyCheck(w http.ResponseWriter, r *http.Request) { 
    log.Printf("ReadyCheck %s\n", r.Body)
    respondwithJSON(w, http.StatusOK, map[string]string{"message": "ready"})
}

func AliveCheck(w http.ResponseWriter, r *http.Request) { 
    log.Printf("ReadyCheck %s\n", r.Body)
    respondwithJSON(w, http.StatusOK, map[string]string{"message": "alive"})
}


func DoInstall(nodename string, data string){
    log.Printf("DoInstall: %s - %s",nodename,data)
}


// Install a New Machine
func InstallNode(w http.ResponseWriter, r *http.Request) { 
    log.Printf("InstallNode: %s\n",r.Body,)
        body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
    v := string(body)
    log.Printf("JSON: %s\n",v)
    hostname := GetLabel(v,`kubernetes\.io/hostname`)
    go DoInstall(hostname, v)
    respondwithJSON(w, http.StatusCreated, map[string]string{"message": "successfully created"})
}

// Update the node
func UpdateNode(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    log.Printf("Update Node: id: %s - %s\n", id, r.Body)
    respondwithJSON(w, http.StatusOK, map[string]string{"message": "update successfully"})

}

// Uninstall a node
func UninstallNode(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    log.Printf("Uninstall Node: id:%s %s\n", id, r.Body)
    respondwithJSON(w, http.StatusOK, map[string]string{"message": "update successfully"})

}

func EnableRestServices() {
        r := routers()
	http.ListenAndServe(":8951", r)
}

// respondwithError return error message
func respondWithError(w http.ResponseWriter, code int, msg string) {
    respondwithJSON(w, code, map[string]string{"message": msg})
}

// respondwithJSON write json response format
func respondwithJSON(w http.ResponseWriter, code int, payload interface{}) {
    response, _ := json.Marshal(payload)
    fmt.Println(payload)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(response)
}


func GetLabel(v string,l string) string{
    result := gjson.Get(v,"labels.#." + l)
    x := result.String()
    x = strings.Replace(x, "[", "", -1)
    x = strings.Replace(x, "]", "", -1)
    x = strings.Replace(x, `"`, "", -1)
    return x
}
 
func GetAnnotation(v string,l string) string{
    result := gjson.Get(v,"annotations.#." + l)
    x := result.String()
    x = strings.Replace(x, "[", "", -1)
    x = strings.Replace(x, "]", "", -1)
    x = strings.Replace(x, `"`, "", -1)
    return x
}

func GetSetting(v string,l string) string{
    result := gjson.Get(v,"settings.#." + l)
    x := result.String()
    x = strings.Replace(x, "[", "", -1)
    x = strings.Replace(x, "]", "", -1)
    x = strings.Replace(x, `"`, "", -1)
    return x
}


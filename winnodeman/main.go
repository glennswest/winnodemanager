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
         "os"
         "encoding/json"
         "fmt"
         "log"
         "time"
)

var builddate string
var gitversion string 

var router *chi.Mux

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



type program struct{}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}
func (p *program) run() {
	// Do work here
        go EnableRestServices()
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	<-time.After(time.Second * 13)
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "winnodeman",
		DisplayName: "WindowsNodeManager",
		Description: "OpenShift 4.x Windows Node Manager - Handles install update and monitoring",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 {
		err = service.Control(s, os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		return
	}

        os.MkdirAll("/Program Files/WindowsNodeManager/logs",0755)
        f, err := os.OpenFile("/Program Files/WindowsNodeManager/logs/winnodeman.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
        if err != nil {
            log.Fatalf("error opening file: %v", err)
            }
        defer f.Close()

        log.SetOutput(f)
        log.Println("winnodemanager restarted - Version: %s - Build Data: %s",gitversion,builddate)
	err = s.Run()
	if err != nil {
		log.Fatalf("Cannot Run: %v",err)
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


func DownloadFile(theurl string, filepath string) error{
// Create the file
  log.Printf("DownloadFile: %s -> %s\n",theurl,filepath)
  out, err := os.Create(filepath)
  if err != nil  {
    log.Printf("Cannot Create File: %v\n",err)
    return err
  }
  defer out.Close()

  // Get the data
  resp, err := http.Get(theurl)
  if err != nil {
    log.Printf("Cannot Get File: %v\n",err)
    return err
  }
  defer resp.Body.Close()

  // Check server response
  if resp.StatusCode != http.StatusOK {
    log.Printf("Bad Response downloading file: %s\n",resp.Status)
    return fmt.Errorf("bad status: %s", resp.Status)
  }

  // Writer the body to file
  _, err = io.Copy(out, resp.Body)
  if err != nil  {
    log.Printf("Cannot Copy File: %v\n",err)
    return err
  }
  return nil

}

func ReadFile(thepath string) string {
    b, err := ioutil.ReadFile(thepath) // just pass the file name
    if err != nil {
        log.Print(err)
        return ""
    }
    str := string(b)
   return str
}
func DoInstall(nodename string, data string){
    basepath := "/Program` Files/WindowsNodeManager"
    os.MkdirAll(basepath + "/settings",0700)
    os.MkdirAll(basepath + "/content",0700)
    log.Printf("DoInstall: %s - %s",nodename,data)
    urlbase := GetSetting(data,"wmmurl")
    template := GetSetting(data,"template")
    urltemplate := "http://" + urlbase + template
    templatepath := basepath + "/settings/template.json"
    log.Printf("Getting template: %s\n",urltemplate)
    err := DownloadFile(urltemplate,templatepath)
    if err != nil {
       log.Printf("Cannot download template")
       return
       }
    tdata := ReadFile(templatepath)
    log.Printf("Template: %s\n",tdata)
    gjson.ForEachLine(tdata, func(line gjson.Result) bool{
          component := line.String()
          cpath := basepath + "/content/" + component
          curl  := urlbase + "/content/" + component
          err := DownloadFile(curl,cpath)
          if err != nil {
             log.Printf("Cannot Download %s - %v\n",component,err)
             return false
             }
          return true
          })
    log.Printf("All Components Downloaded\n")

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


package main

import (
	"net/http"
        "io"
	"io/ioutil"
	"github.com/go-chi/chi"
        "github.com/go-chi/chi/middleware"
        "github.com/tidwall/gjson"
        "github.com/glennswest/libignition/ignition"
	"github.com/kardianos/service"
        . "github.com/glennswest/go-sshclient"
        b64 "encoding/base64"
	"gopkg.in/natefinch/lumberjack.v2"
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
     router.Post("/data/{filename}", StoreData)
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
    //router.Use(middleware.URLFormat)
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
        log.SetOutput(&lumberjack.Logger{
                Filename:   "/Program Files/WindowsNodeManager/logs/winnodeman.log",
                MaxSize:    1, // megabytes
                MaxBackups: 6,
                MaxAge:     1, // days
                Compress:   true, // disabled by default
                })
        log.Printf("winnodemanager restarted - Version: %s - Build Data: %s",gitversion,builddate)
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
    basepath := "/Program Files/WindowsNodeManager"
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
    result := gjson.Get(tdata, "packages")
    for _, name := range result.Array() {
          component := name.String()
          log.Printf("Processing Component: %s\n",component)
          cpath := basepath + "/content/" + component + ".ign"
          curl  := "http://" + urlbase + "/content/" + component + ".ign"
          err := DownloadFile(curl,cpath)
          if err != nil {
             log.Printf("Cannot Download %s - %v\n",component,err)
             }
          }
    log.Printf("All Components Downloaded\n")
    log.Printf("Starting deployment of ignition files\n")
    for _, name := range result.Array() {
          component := name.String()
          log.Printf("Using Ignition to Deploy: %s\n",component)
          cpath := basepath + "/content/" + component + ".ign"
          status := ignition.Parse_ignition_file(cpath)
          if (status != 0){
             log.Printf("Failed Deployment for component %s\n",component)
             }
          }
    log.Printf("Starting Command Execution from Components\n")
    for _, name := range result.Array() {
          component := name.String()
          log.Printf("Pulling metadata for component: %s\n",component)
          mpath := "/bin/metadata/" + component + ".metadata"
          md := ReadFile(mpath)
          if (len(md) > 0){
             process_install_metadata(nodename,data,component,md)
             } else {
             log.Printf("Missing metadata for component: %s\n",component)
             }
          }

}

func process_install_metadata(nodename string,d string,cname string,md string){
     log.Printf("Processing %s metadata\n",cname)
     description := gjson.Get(md,"description").String()
     imessage := gjson.Get(md,"install_message").String()
     url      := gjson.Get(md,"package_url").String()
     if (len(description) > 0){
        log.Printf("Component: %s\n", description)
        }
     if (len(imessage) > 0){
        log.Printf("%s\n",imessage)
        }
     if (len(url) > 0){
        log.Printf("Using content from: %s\n",url)
        }
     lprecmds := gjson.Get(md,"install.lprecmds").Array()
     commands := gjson.Get(md,"install.commands").Array()
     lpstcmds := gjson.Get(md,"install.lpstcmds").Array()
     process_master_commands(lprecmds,nodename,d,cname,md,"lprecmds")
     process_local_commands(commands,nodename,d,cname,md,"commands")
     process_master_commands(lpstcmds,nodename,d,cname,md,"lpstcmds")
}

func trimQuotes(s string) string {
    if len(s) >= 2 {
        if s[0] == '"' && s[len(s)-1] == '"' {
            return s[1 : len(s)-1]
        }
    }
    return s
}

func getkeyvalue(d string) (string, string) {
    x := strings.Replace(d,"{","",-1)
    x = strings.Replace(x,"}","",-1)
    result := strings.Split(x, ":")
    k := trimQuotes(result[0])
    v := trimQuotes(result[1])
    return k,v
}

func processvars(d string,vtype string) []string{
var r[] string

    result := gjson.Get(d, vtype)
    result.ForEach(func(key, value gjson.Result) bool {
       k,v := getkeyvalue(value.String())
       switch(k){
          case "user":
                break;
          case "password":
                break;
          case "sshuser":
                break;
          case "sshkey":
                break;
          default:
                k = strings.Replace(k,".","_",-1)
                k = strings.Replace(k,"/","_",-1)
                l := "export " + k + "=" + `"` + v + `"`
                r = append(r,l)
                }
       return true
       })
    return(r)

}

func envars(d string) []string{
var  r[] string

    r  = processvars(d,"settings")
    l := processvars(d,"labels")
    a := processvars(d,"annotations")
    r  = append(r,l...)
    r  = append(r,a...)
    return(r)
}


func process_master_commands(cmds []gjson.Result,nodename string,d string,cname string,md string,itype string){
var script[] string

    l := len(cmds)
    if (l == 0){
       return
       }
    log.Printf("Processing Master Commands - Qty %d\n",l)
    env := envars(d)
    script = append(script,env...)
    log.Printf("script\n")
    for _, cmd := range cmds {
          script = append(script,cmd.String())
          log.Printf("    %s\n",cmd.String())
          }

    host := GetSetting(d,"master")
    log.Printf("Master host: %s\n",host)
    username := GetSetting(d,"sshuser")
    os.MkdirAll("/Program Files/WindowsNodeManager/install/",0700)
    sshkey_path := "/Program Files/WindowsNodeManager/install/id"
    sshkeyb64 := GetSetting(d,"sshkey")
    sshkeybytes, _ := b64.StdEncoding.DecodeString(sshkeyb64)
    ioutil.WriteFile(sshkey_path, sshkeybytes, 0600)
    SshCommand(host,username,sshkey_path,script)
    //os.Remove(sshkey_path)
}

func process_local_commands(cmds []gjson.Result,nodename string,d string,cname string,md string,itype string){
    l := len(cmds)
    if (l == 0){
       return
       }
    log.Printf("Processing Local Commands - Qty %d\n",l)
    //os.MkdirAll("/Program Files/WindowsNodeManager/install/" + cname + "/" + itype,0755)
}

func StoreData(w http.ResponseWriter, r *http.Request) {
        log.Printf("StoreData: %s\n",r.Body,)
        // Security FixMe: Need to block . .. and / within
        filename := chi.URLParam(r, "filename")
        body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
        if err != nil {
                panic(err)
        }
        if err := r.Body.Close(); err != nil {
                panic(err)
        }
       dpath := "/Program Files/WindowsNodeManager/data/"
       os.MkdirAll(dpath,0700)
       ioutil.WriteFile(dpath + filename, body, 0700)
       respondwithJSON(w, http.StatusCreated, map[string]string{"message": "successfully created"})

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

func SshCommand(host string,username string,keypath string,cmds []string) []string{
var sshout[] string

    hp := host
    if (strings.ContainsAny(hp,":") == false){
       hp = host + ":22"
       }
    // Keypath should be pathname of private key
    client, err := DialWithKey(hp, username, keypath)
    if (err != nil){
       log.Printf("SSHCommand: Cannot Connect to %s - %v\n",host,err)
       return(sshout)
       }
    defer client.Close()
    for _, c := range cmds {
       log.Printf("SSH: %s\n",c)
       out, _ := client.Cmd(c).Output()
       log.Printf("ssh: %s\n",string(out))
       sshout = append(sshout,string(out))
       }
    return(sshout)
}




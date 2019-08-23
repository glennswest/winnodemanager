package main

import (
	"net/http"
        "io"
	"io/ioutil"
        "net"
	"github.com/go-chi/chi"
        "github.com/go-chi/chi/middleware"
        "github.com/tidwall/sjson"
        "github.com/tidwall/gjson"
        "github.com/glennswest/libignition/ignition"
	"github.com/kardianos/service"
        . "github.com/glennswest/go-sshclient"
        b64 "encoding/base64"
	"gopkg.in/natefinch/lumberjack.v2"
        "github.com/capnspacehook/taskmaster"
        "github.com/glennswest/libpowershell/pshell"
        "strings"
         "os"
         "encoding/json"
         "fmt"
         "log"
         "time"
)

const Basepath string = "/Program Files/WindowsNodeManager"
var builddate string
var gitversion string 
var TheLog *lumberjack.Logger

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
        go restart_install()
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
        TheLog := &lumberjack.Logger{
                Filename:   "/Program Files/WindowsNodeManager/logs/winnodeman.log",
                MaxSize:    1, // megabytes
                MaxBackups: 6,
                MaxAge:     1, // days
                Compress:   true, // disabled by default
                }
        log.SetOutput(TheLog)
        log.Printf("Winnodemanager restarted - version: %s - build Data: %s",gitversion,builddate)
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
  log.Printf("Download File: %s -> %s\n",theurl,filepath)
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
        //log.Print(err)
        return ""
    }
    str := string(b)
   return str
}

func WriteFile(thepath string,data string){
     ioutil.WriteFile(thepath,[]byte(data), 0600)
}

func DoInstall(nodename string, data string){
    // Make sure we have master ip
    master := GetSetting(data,"master")
    masterip,_ := net.LookupHost(master)
    data = ArAdd(data,"settings","masterip",masterip[0])

    os.MkdirAll(Basepath + "/state",0700)
    os.MkdirAll(Basepath + "/settings",0700)
    os.MkdirAll(Basepath + "/settings/env",0700)
    os.MkdirAll(Basepath + "/settings/env/settings",0700)
    os.MkdirAll(Basepath + "/settings/env/labels",0700)
    os.MkdirAll(Basepath + "/settings/env/annotations",0700)
    
    os.MkdirAll(Basepath + "/content",0700)
    log.Printf("DoInstall: %s ",nodename)
    win_writevars(data)
    urlbase := GetSetting(data,"wmmurl")
    template := GetSetting(data,"template")
    urltemplate := urlbase + template
    templatepath := Basepath + "/settings/template.json"
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
          cpath := Basepath + "/content/" + component + ".ign"
          curl  := urlbase + "/content/" + component + ".ign"
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
          cpath := Basepath + "/content/" + component + ".ign"
          status := ignition.Parse_ignition_file(cpath,"")
          if (status != 0){
             log.Printf("Failed Deployment for component %s\n",component)
             }
          }
    spath := Basepath + "/state/"
    WriteFile(spath + "nodename.state",nodename)
    WriteFile(spath + "data.state",data)
    WriteFile(spath + "install.state","running")
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
    WriteFile(spath + "install.state","done")
    log.Printf("Install Complete\n")
}
 
// Handle the install after restart
func restart_install(){
    spath := Basepath + "/state/"
    state := ReadFile(spath + "install.state")
    switch(state){
       case "":
            log.Printf("restart_install: No Install pending\n")
            return
            break
       case "running":
            log.Printf("restart_install: Continue Install\n")

       case "done":
            log.Printf("restart_install: Install Completed - Not Restarting\n")
            return
            break
       default:
            log.Printf("restart_install: Unknown State - Not retrying\n")
            return
            break
       }
    log.Printf("Waiting for system to stablize\n")
    time.Sleep(90 * time.Second)
    log.Printf("Install process proceeding\n")
    nodename := ReadFile(spath + "nodename.state")
    data     := ReadFile(spath + "data.state")
    templatepath := Basepath + "/settings/template.json"
    tdata := ReadFile(templatepath)
    result := gjson.Get(tdata, "packages")
    for _, name := range result.Array() {
          component := name.String()
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
     spath := Basepath + "/state/comp_" + cname
     state := ReadFile(spath)
     switch(state){
        case "":   // Empty or no file = Not Started
             break
        case "done":
             log.Printf("Skipping completed component %s\n",cname)
             return
             break;
         default:
             break
         }
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
     WriteFile(spath,"done")  // So if we reboot it will skip it next time
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
          case "wmmurl":
               break;
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
func win_processvars(d string,vtype string) []string{
var r[] string

    result := gjson.Get(d, vtype)
    result.ForEach(func(key, value gjson.Result) bool {
       k,v := getkeyvalue(value.String())
       if (len(v) < 128){
       switch(k){
          case "wmmurl":
               break;
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
                //  $Env:kubernetes_io_hostname
                l := "$Env:" + k + "=" + `"` + v + `"`
                r = append(r,l)
                }
          }
       return true
       })
    return(r)

}

func IsBase64(s string) bool {
	_, err := b64.StdEncoding.DecodeString(s)
	return err == nil
}

func win_savevars(d string,vtype string) []string{
var r[] string

    envpath := Basepath + "/settings/env/" + vtype
    result := gjson.Get(d, vtype)
    result.ForEach(func(key, value gjson.Result) bool {
       k,v := getkeyvalue(value.String())
       switch(k){
          case "wmmurl":
               break;
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
                thepath := envpath + "/" + k
                vx := v
                if (IsBase64(v)){
                   vxb,_ := b64.StdEncoding.DecodeString(v)
                   vx = string(vxb)
                   }
                WriteFile(thepath,vx)
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


func win_envars(d string) []string{
var  r[] string

    r  = win_processvars(d,"settings")
    l := win_processvars(d,"labels")
    a := win_processvars(d,"annotations")
    r  = append(r,l...)
    r  = append(r,a...)
    return(r)
}

func win_writevars(d string){

    win_savevars(d,"settings")
    win_savevars(d,"labels")
    win_savevars(d,"annotations")
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
    for _, cmd := range cmds {
          script = append(script,cmd.String())
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

func wait_for_file(filename string){

        total_time := 0;
        time_limit := 60 * 15 // 15 Minutes
	for {
          if (fileExists(filename)){
             return
             }
         time.Sleep(2 * time.Second)
         total_time = total_time + 2
         if (total_time > time_limit){
            log.Printf("Timout waiting for done file %s\n",filename)
            return
            }
         }



}

func fileExists(filename string) bool {
    info, err := os.Stat(filename)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}


func schedule_task(thepath string,thename string){

        taskService, _ := taskmaster.Connect("", "", "", "")
        defer taskService.Disconnect()

        newTaskDef := taskService.NewTaskDefinition()

        cmd :=  "-windowstyle hidden -ExecutionPolicy Unrestricted -NonInteractive " + thepath  + " *> C:\\k\\logs\\" + thename + ".out"
        newTaskDef.AddExecAction("powershell",cmd,"","")
        newTaskDef.RegistrationInfo.Author = "WinNodeManager"
        newTaskDef.RegistrationInfo.Description = thename
        newTaskDef.Principal.UserID = "SYSTEM"
        taskpath := "\\WinNodeManager\\" + thename

        log.Printf("Taskpath: %s\n",taskpath)
        newTask, _, err  := taskService.CreateTask(taskpath, newTaskDef, true)
        if (err != nil){
           panic(err)
           }
        log.Printf("Run it\n")
        running_task, err := newTask.Run([]string{thename})
        log.Printf("ScheduleJob: %s\n",cmd)
        log.Printf("Task: %v\n",running_task)
        if (err != nil){
           log.Printf("Error: %v",err)
           }
}


func process_local_commands(cmds []gjson.Result,nodename string,d string,cname string,md string,itype string){
    l := len(cmds)
    if (l == 0){
       return
       }
    username := GetSetting(d,"user")
    password := GetSetting(d,"password")
    hostip   := GetAnnotation(d,"host/ip")
    //log.Printf("Username: %s Password: %s\n:",username,password)
    donepath := "/k/tmp/" + cname + ".done"
    pshell.SetRemoteMode(hostip,username,password)
    log.Printf("Processing Local Commands - Qty %d\n",l)
    env := win_envars(d)
    pshellcmd := ""
    for _, ln := range env {
          if (len(pshellcmd) > 0){
             pshellcmd = pshellcmd + ";"
             }
          pshellcmd = pshellcmd + ln
          }
    scheduled_job := false
    if (cmds[0].String() == "#job"){
       scheduled_job = true
       }
    for _, ln := range cmds {
          theline := ln.String()
          if (theline[0] == '#'){
             theline = ""
             }
          if (len(pshellcmd) > 0){
             pshellcmd = pshellcmd + ";"
             }
          pshellcmd = pshellcmd + theline
          }
     pshellcmd = pshellcmd + "; echo $null >> " + donepath
     thepath := "/bin/run_" + cname + ".ps1"
     WriteFile(thepath,pshellcmd)
     cmd := thepath + " *> " + "/k/logs/run_" + cname + ".out"
     spcmd := "Start-Process \"powershell\" -args \"" + cmd + "\" -NoNewWindow -Wait"
     if (scheduled_job == true){
        schedule_task(thepath,cname)
      } else {
        pshell.Powershell(spcmd)
      }
     wait_for_file(donepath)
}

func StoreData(w http.ResponseWriter, r *http.Request) {
        // Security FixMe: Need to block . .. and / within
        filename := chi.URLParam(r, "filename")
        log.Printf("StoreData: %s\n",filename)
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
    //TheLog.Rotate()
    log.Printf("InstallNode: %s\n",r.Body,)
        body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
    v := string(body)
    //log.Printf("JSON: %s\n",v)
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
    os.RemoveAll(Basepath + "/state")
    os.RemoveAll(Basepath + "/settings")
    os.RemoveAll(Basepath + "/content")
    os.RemoveAll(Basepath + "/install")
    os.RemoveAll(Basepath + "/data")
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

func ArAdd(d string,aname string,v1 string,v2 string) string{
      s := `{"` + v1 + `":"` + v2 + `"}`
      a := aname + ".-1"
      d,_ = sjson.SetRaw(d,a,s)
      return d
      }

func SshCommand(host string,username string,keypath string,cmds []string) string{

    hp := host
    if (strings.ContainsAny(hp,":") == false){
       hp = host + ":22"
       }
    // Keypath should be pathname of private key
    client, err := DialWithKey(hp, username, keypath)
    if (err != nil){
       log.Printf("SSHCommand: Cannot Connect to %s - %v\n",host,err)
       return("")
       }
    defer client.Close()
    lc := ""
    for _, c := range cmds {
       if (len(lc) > 0){
          lc = lc + "; "
          }
       lc = lc + c
       }
    log.Printf("%s\n",lc)
    out, err := client.Cmd(lc).SmartOutput()
    if (err != nil){
          log.Printf("SSHCommand: Cannot send cmd: %v\n",err)
          }
    log.Printf("ssh: %s\n",string(out))
    return(string(out))
}


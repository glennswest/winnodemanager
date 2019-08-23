package main

import (
        "github.com/capnspacehook/taskmaster"
        "github.com/rickb777/date/period"
        "time"
        "log"
)

func schedule_task(thepath string,thename string){

        taskService, _ := taskmaster.Connect("", "", "System", "")
        defer taskService.Disconnect()

        newTaskDef := taskService.NewTaskDefinition()

        newTaskDef.AddExecAction("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",thepath,"", "")
        newTaskDef.RegistrationInfo.Author = "WinNodeManager"
        newTaskDef.RegistrationInfo.Description = thename
        newTaskDef.AddTimeTrigger(period.NewHMS(0, 0, 0), time.Now().Add(5*time.Second))
        taskpath := "\\WinNodeManager\\" + thename

        _, _, _ = taskService.CreateTask(taskpath, newTaskDef, true)
}

func main(){
        log.Printf("Getting ready\n")
	schedule_task("\bin\run_node_2.0.0.ps1","node_2.0.0")
        log.Printf("Done")
}


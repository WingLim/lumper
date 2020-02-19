package main

import (
	log "github.com/sirupsen/logrus"
	"lumper/cgroups/subsystems"
	"os"
	"lumper/container"
	"strings"
	"lumper/cgroups"
)

func Run(tty bool, cmdArray []string, res * subsystems.ResourceConfig, containerName string)  {
	parent, writePipe := container.NewParentProcess(tty)
	if parent == nil {
		log.Errorf("new parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	// 创建 Cgroup Manager
	cgroupManager := cgroups.NewCgroupManager("lumper-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)
	sendInitCommand(cmdArray, writePipe)
	if tty {
		parent.Wait()
	}
	mntURL := "/root/mnt/"
	rootURL := "/root/"
	container.DeleteWorkSpace(rootURL, mntURL)
}

func sendInitCommand(cmdArray []string, writePipe *os.File)  {
	command := strings.Join(cmdArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

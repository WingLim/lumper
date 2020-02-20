package main

import (
	log "github.com/sirupsen/logrus"
	"strings"
	"os/exec"
	"os"
	_ "lumper/nsenter"
)

const ENV_EXEC_PID = "lumper_pid"
const ENV_EXEC_CMD = "lumper_cmd"

func ExecContainer(containerName string, cmdArray []string)  {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		log.Errorf("get container pid by name %s error %v", containerName, err)
		return
	}
	cmdStr := strings.Join(cmdArray, " ")
	log.Infof("container pid %s", pid)
	log.Infof("command %s", cmdStr)
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)

	if err := cmd.Run(); err != nil {
		log.Errorf("exec container %s error %v", containerName, err)
	}
}

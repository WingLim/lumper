package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"strings"
)

func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("run container get user command error, cmdArray is nil")
	}
	// MS_NOEXEC 不允许运行其他程序，MS_NOSUID 不允许 set-user-ID 或 set-group-ID，MS_NODEV 不允许访问设备
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	// 寻找命令的绝对路径
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		log.Errorf("exec loop path error %v", err)
	}
	log.Infof("find path %s", path)
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		log.Errorf(err.Error())
	}
	return nil
}

// 读取用户命令
func readUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		log.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")

}

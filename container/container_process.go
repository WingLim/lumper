package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
)

var (
	RUNNING 			string = "running"
	STOP 				string = "stopped"
	EXIT 				string = "exited"
	DefaultInfoLocation string = "/var/run/lumper/%s/"
	ConfigName 			string = "config.json"
	ContainerLogFile 	string = "container.log"
	RootUrl 			string = "/root/"
	MntUrl 				string = "/root/mnt/%s/"
	WriteLayerUrl 		string = "/root/writeLayer/%s/"
)

type ContainerInfo struct {
	Pid         string `json:"pid"` // 容器 init 进程在宿主机上的 PID
	Id          string `json:"id"`  // 容器 Id
	Name        string `json:"name"`  // 容器名
	Command     string `json:"command"`    // 容器内 init 运行命令
	CreatedTime string `json:"createTime"` // 创建时间
	Status      string `json:"status"`     // 容器状态
	Volume      string `json:"volume"` // 容器数据卷
}

// 创建一个父进程
func NewParentProcess(tty bool, containerName , volume , imageName string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		log.Errorf("new pipe error %v", err)
		return nil, nil
	}
	cmd := exec.Command("/proc/self/exe", "init")
	// 克隆一个新进程，使用 namespace 隔离新进程和外部环境
	cmd.SysProcAttr = &unix.SysProcAttr{
		Cloneflags: unix.CLONE_NEWUTS | unix.CLONE_NEWPID | unix.CLONE_NEWNS | unix.CLONE_NEWNET | unix.CLONE_NEWIPC,
	}
	// 如果指定 tty 参数，则将当前进程输入输出导入到标准输入输出上
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(dirUrl, 0622); err != nil {
			log.Errorf("mkdir %s error %v", dirUrl, err)
			return nil, nil
		}
		stdLogFilePath := dirUrl + ContainerLogFile
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			log.Errorf("create file %s error %v", stdLogFilePath, err)
			return nil, nil
		}
		cmd.Stdout = stdLogFile
	}
	// 传入管道文件读取端的句柄
	cmd.ExtraFiles = []*os.File{readPipe}
	NewWorkSpace(volume, containerName, imageName)
	cmd.Dir = fmt.Sprintf(MntUrl, containerName)
	return cmd, writePipe
}

// 创建一个管道
func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}

package container

import (
	"golang.org/x/sys/unix"
	"os/exec"
	"os"
	log "github.com/sirupsen/logrus"
)

// 创建一个父进程
func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
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
	}
	// 传入管道文件读取端的句柄
	cmd.ExtraFiles = []*os.File{readPipe}
	cmd.Dir = "/root/busybox"
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
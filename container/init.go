package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("run container get user command error, cmdArray is nil")
	}

	setUpMount()

	// 寻找命令的绝对路径
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		log.Errorf("exec loop path error %v", err)
	}
	log.Infof("find path %s", path)
	if err := unix.Exec(path, cmdArray[0:], os.Environ()); err != nil {
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

func setUpMount()  {
	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("get current localtion error %v", err)
		return
	}
	log.Infof("current location is %s", pwd)

	unix.Mount("", "/", "", unix.MS_PRIVATE | unix.MS_REC, "")

	pivotRoot(pwd)

	// MS_NOEXEC 不允许运行其他程序，MS_NOSUID 不允许 set-user-ID 或 set-group-ID，MS_NODEV 不允许访问设备
	defaultMountFlags := unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_NODEV
	unix.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	unix.Mount("tmpfs", "/dev", "tmpfs", unix.MS_NOSUID | unix.MS_STRICTATIME, "mode=755")
}

func pivotRoot(root string) error {
	// 重新挂载 root
	if err := unix.Mount(root, root, "bind", unix.MS_BIND | unix.MS_REC, ""); err != nil {
		return fmt.Errorf("mount rootfs to itself error %v", err)
	}
	// 创建 rootfs/.pivot_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}
	// pivot_root 到新的 rootfs，老的 root 挂载在 rootfs/.pivot_root
	if err := unix.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root error %v", err)
	}
	// 修改当前工作目录到根目录
	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("chdir to / error %v", err)
	}
	// 卸载 rootfs/.pivot_root
	pivotDir = filepath.Join("/", ".pivot_root")
	if err := unix.Unmount(pivotDir, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir error %v", err)
	}
	return os.Remove(pivotDir)
}

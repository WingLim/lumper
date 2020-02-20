package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"strings"
)

var (
	RUNNING string = "running"
	STOP string = "stopped"
	EXIT string = "exited"
	DefaultInfoLocation string = "/var/run/lumper/%s/"
	ConfigName string = "config.json"
	ContainerLogFile string = "container.log"
)

type ContainerInfo struct {
	Pid         string `json:"pid"` // 容器 init 进程在宿主机上的 PID
	Id          string `json:"id"`  // 容器 Id
	Name        string `json:"name"`  // 容器名
	Command     string `json:"command"`    // 容器内 init 运行命令
	CreatedTime string `json:"createTime"` // 创建时间
	Status      string `json:"status"`     // 容器状态
}

// 创建一个父进程
func NewParentProcess(tty bool, containerName string, volume string) (*exec.Cmd, *os.File) {
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
	mntURL := "/root/mnt/"
	rootURL := "/root/"
	NewWorkSpace(rootURL, mntURL, volume)
	cmd.Dir = mntURL
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

func NewWorkSpace(rootURL string, mntURL string, volume string) {
	CreateReadOnlyLayer(rootURL)
	CreateWriteLayer(rootURL)
	CreateMountPoint(rootURL, mntURL)
	// 存在 volume 则挂载
	if volume != "" {
		volumeUrls := volumeUrlExtract(volume)
		length := len(volumeUrls)
		if length == 2 && volumeUrls[0] != "" && volumeUrls[1] != "" {
			MountVolume(rootURL, mntURL, volumeUrls)
			log.Infof("%q", volumeUrls)
		} else {
			log.Infof("volume parameter input is not correct")
		}
	}
}

// 解压 busybox.tar 到 busybox 目录下，作为容器的只读层
func CreateReadOnlyLayer(rootURL string) {
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busybox.tar"
	exist, err := PathExists(busyboxURL)
	if err != nil {
		log.Infof("fail to judge whether dir %s exists %v", busyboxURL, err)
	}
	if exist == false {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			log.Errorf("mkdir dir %s error %v", err)
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			log.Errorf("unTar dir %s error %v", busyboxTarURL, err)
		}
	}
}

// 创建 writeLayer 文件夹作为容器唯一的可写层
func CreateWriteLayer(rootURL string) {
	writeUrl := rootURL + "writeLayer/"
	if err := os.Mkdir(writeUrl, 0777); err != nil {
		log.Errorf("mkdir dir %s error %v", writeUrl, err)
	}
}

func CreateMountPoint(rootURL string, mntURL string) {
	//创建 mnt 文件夹作为挂载点
	if err := os.Mkdir(mntURL, 0777); err != nil {
		log.Errorf("mkdir dir %s error %v", mntURL, err)
	}
	// 把 writeLayer 目录和 busybox 目录挂载到 mnt 目录下
	dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("mount dirs error %v", err)
	}
}

func DeleteWorkSpace(rootURL string, mntURL string, volume string) {
	if (volume != "") {
		volumeUrls := volumeUrlExtract(volume)
		length := len(volumeUrls)
		if(length == 2 && volumeUrls[0] != "" && volumeUrls[1] != "") {
			DeleteMountPointWithVolume(rootURL, mntURL, volumeUrls)
		} else {
			DeleteMountPoint(rootURL, mntURL)
		}
	} else {
		DeleteMountPoint(rootURL, mntURL)
	}
	DeleteWriteLayer(rootURL)
}

// 删除挂载点
func DeleteMountPoint(rootURL string, mntURL string) {
	// 卸载挂载点
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("umount error %v", err)
	}
	// 删除 mnt 文件夹
	if err := os.RemoveAll(mntURL); err != nil {
		log.Errorf("remove dir %s error %v", mntURL, err)
	}
}

// 卸载 volume 和删除挂载点
func DeleteMountPointWithVolume(rootURL string, mntURL string, volumeUrls []string)  {
	containerUrl := mntURL + volumeUrls[1]
	// 卸载 volume
	cmd := exec.Command("umount", containerUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("umount volume failed %v", err)
	}
	// 卸载挂载点
	cmd = exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("umount mountpoing failed %v", err)
	}
	// 删除 mnt 文件夹
	if err := os.RemoveAll(mntURL); err != nil {
		log.Infof("remove mountpoint dir %s error", mntURL, err)
	}

}

func DeleteWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.RemoveAll(writeURL); err != nil {
		log.Errorf("remove dir %s error %v", writeURL, err)
	}
}

// 挂载 volume
func MountVolume(rootURL string, mntURL string, volumeUrls []string) {
	// 创建宿主机文件目录
	parentUrl := volumeUrls[0]
	if err := os.Mkdir(parentUrl, 0777); err != nil {
		log.Errorf("mkdir parent dir %s error %v", parentUrl, err)
	}
	// 在容器文件系统中创建挂载点
	containerUrl := volumeUrls[1]
	containerVolumeUrl := mntURL + containerUrl
	if err := os.Mkdir(containerVolumeUrl, 0777); err != nil {
		log.Errorf("mkdir container dir %s error %v", containerVolumeUrl, err)
	}
	// 把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("mount volume failed %v", err)
	}
}

// 判断文件路径是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// 解析 volume 字符串
func volumeUrlExtract(volume string) ([]string) {
	var volumeUrls []string
	volumeUrls = strings.Split(volume, ":")
	return  volumeUrls
}

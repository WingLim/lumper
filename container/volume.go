package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	log "github.com/sirupsen/logrus"
)

func NewWorkSpace(volume ,imageName, containerName string) {
	CreateReadOnlyLayer(imageName)
	CreateWriteLayer(containerName)
	CreateMountPoint(containerName, imageName)
	// 存在 volume 则挂载
	if volume != "" {
		volumeUrls := volumeUrlExtract(volume)
		length := len(volumeUrls)
		if length == 2 && volumeUrls[0] != "" && volumeUrls[1] != "" {
			MountVolume(volumeUrls, containerName)
			log.Infof("%q", volumeUrls)
		} else {
			log.Infof("volume parameter input is not correct")
		}
	}
}

// 解压 busybox.tar 到 busybox 目录下，作为容器的只读层
func CreateReadOnlyLayer(imageName string) {
	// 镜像解压路径
	folderUrl := RootUrl + imageName + "/"
	// 镜像压缩包
	imageUrl := RootUrl + imageName + ".tar"
	exist, err := PathExists(folderUrl)
	if err != nil {
		log.Infof("fail to judge whether dir %s exists %v", folderUrl, err)
	}
	if !exist {
		if err := os.Mkdir(folderUrl, 0622); err != nil {
			log.Errorf("mkdir %s error %v",folderUrl, err)
		}
		if _, err := exec.Command("tar", "-xvf", imageUrl, "-C", folderUrl).CombinedOutput(); err != nil {
			log.Errorf("unTar dir %s error %v", folderUrl, err)
		}
	}
}

// 创建 writeLayer 文件夹作为容器唯一的可写层
func CreateWriteLayer(containerName string) {
	writeUrl := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.Mkdir(writeUrl, 0777); err != nil {
		log.Errorf("mkdir dir %s error %v", writeUrl, err)
	}
}

func CreateMountPoint(containerName, imageName string) {
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	//创建 mnt 文件夹作为挂载点
	if err := os.Mkdir(mntUrl, 0777); err != nil {
		log.Errorf("mkdir dir %s error %v", mntUrl, err)
	}
	tmpWritLayer := fmt.Sprintf(WriteLayerUrl, containerName)
	tmpImageLocation := RootUrl + imageName
	// 把 writeLayer 目录和 busybox 目录挂载到 mnt 目录下
	dirs := "dirs=" + tmpWritLayer + "writeLayer:" + tmpImageLocation
	if _, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntUrl).CombinedOutput(); err != nil {
		log.Errorf("mount dirs error %v", err)
	}
}

func DeleteWorkSpace(volume, containerName string) {
	if (volume != "") {
		volumeUrls := volumeUrlExtract(volume)
		length := len(volumeUrls)
		if(length == 2 && volumeUrls[0] != "" && volumeUrls[1] != "") {
			DeleteMountPointWithVolume(volumeUrls, containerName)
		} else {
			DeleteMountPoint(containerName)
		}
	} else {
		DeleteMountPoint(containerName)
	}
	DeleteWriteLayer(containerName)
}

// 删除挂载点
func DeleteMountPoint(containerName string) {
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	// 卸载挂载点
	if _, err := exec.Command("umount", mntUrl).CombinedOutput(); err != nil {
		log.Errorf("umount error %v", err)
	}
	// 删除 mnt 文件夹
	if err := os.RemoveAll(mntUrl); err != nil {
		log.Errorf("remove dir %s error %v", mntUrl, err)
	}
}

// 卸载 volume 和删除挂载点
func DeleteMountPointWithVolume(volumeUrls []string, containerName string)  {
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	containerUrl := mntUrl + volumeUrls[1]
	// 卸载 volume
	if _, err := exec.Command("umount", containerUrl).CombinedOutput(); err != nil {
		log.Errorf("umount volume failed %v", err)
	}
	// 卸载挂载点
	if _, err := exec.Command("umount", mntUrl).CombinedOutput(); err != nil {
		log.Errorf("umount mountpoing failed %v", err)
	}
	// 删除 mnt 文件夹
	if err := os.RemoveAll(mntUrl); err != nil {
		log.Infof("remove mountpoint dir %s error", mntUrl, err)
	}

}

func DeleteWriteLayer(containerName string) {
	writeUrl := fmt.Sprintf(WriteLayerUrl,  containerName)
	if err := os.RemoveAll(writeUrl); err != nil {
		log.Errorf("remove dir %s error %v", writeUrl, err)
	}
}

// 挂载 volume
func MountVolume(volumeUrls []string, containerName string) {
	// 创建宿主机文件目录
	parentUrl := volumeUrls[0]
	if err := os.Mkdir(parentUrl, 0777); err != nil {
		log.Errorf("mkdir parent dir %s error %v", parentUrl, err)
	}
	// 在容器文件系统中创建挂载点
	containerUrl := volumeUrls[1]
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	containerVolumeUrl := mntUrl + containerUrl
	if err := os.Mkdir(containerVolumeUrl, 0777); err != nil {
		log.Errorf("mkdir container dir %s error %v", containerVolumeUrl, err)
	}
	// 把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeUrl)
	// 查看错误
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


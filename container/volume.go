package container

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"strings"
	log "github.com/sirupsen/logrus"
)

func NewWorkSpace(volume , containerName, imageName string) {
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

// 解压镜像压缩包，作为容器的只读层
func CreateReadOnlyLayer(imageName string) error {
	// 镜像解压路径
	folderUrl := fmt.Sprintf(ImageLocation, imageName)
	// 镜像压缩包
	imageUrl := RootUrl + imageName + ".tar"
	exist, err := PathExists(folderUrl)
	if err != nil {
		log.Infof("fail to judge whether dir %s exists %v", folderUrl, err)
		return err
	}
	if !exist {
		if err := os.MkdirAll(folderUrl, 0622); err != nil {
			log.Errorf("mkdir %s error %v",folderUrl, err)
			return err
		}
		if _, err := exec.Command("tar", "-xvf", imageUrl, "-C", folderUrl).CombinedOutput(); err != nil {
			log.Errorf("unTar dir %s error %v", folderUrl, err)
			return err
		}
	}
	return nil
}

// 创建 upper 和 work 文件夹作为容器的可写层
func CreateWriteLayer(containerName string) {
	containerUrl := fmt.Sprintf(Overlay2Location, containerName)
	upperUrl := containerUrl + "upper"
	workUrl := containerUrl + "work"
	if err := os.MkdirAll(upperUrl, 0777); err != nil {
		log.Errorf("mkdir dir %s error %v", upperUrl, err)
	}
	if err := os.MkdirAll(workUrl, 0777); err != nil {
		log.Errorf("mkdir dir %s error %v", workUrl, err)
	}
}

func CreateMountPoint(containerName, imageName string) error {
	containerUrl := fmt.Sprintf(Overlay2Location, containerName)
	mergeUrl := containerUrl + "merged"
	//创建 merged 文件夹作为挂载点
	if err := os.MkdirAll(mergeUrl, 0777); err != nil {
		log.Errorf("mkdir dir %s error %v", mergeUrl, err)
		return err
	}
	upperUrl := containerUrl + "upper"
	workUrl := containerUrl + "work"
	tmpImageLocation := fmt.Sprintf(ImageLocation, imageName)
	// 把 writeLayer 目录和 busybox 目录挂载到 mnt 目录下
	dirs := "lowerdir=" + tmpImageLocation +",upperdir=" + upperUrl + ",workdir=" + workUrl
	if _, err := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, mergeUrl).CombinedOutput(); err != nil {
		log.Errorf("mount dirs error %v", err)
		return err
	}
	return nil
}

func DeleteWorkSpace(volume, containerName, imageName string) {
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
	DeleteContainerFolder(containerName)
}

// 删除挂载点
func DeleteMountPoint(containerName string) error {
	containerUrl := fmt.Sprintf(Overlay2Location, containerName)
	mergedUrl := containerUrl + "merged"
	// 卸载挂载点
	if err := unix.Unmount(mergedUrl, unix.MNT_FORCE) ; err != nil {
		log.Errorf("umount merged floder failed %v", err)
		return err
	}
	// 删除 mnt 文件夹
	if err := os.RemoveAll(mergedUrl); err != nil {
		log.Errorf("remove merged floder %s error %v", mergedUrl, err)
		return err
	}
	return nil
}

// 卸载 volume 和删除挂载点
func DeleteMountPointWithVolume(volumeUrls []string, containerName string) error {
	containerUrl := fmt.Sprintf(Overlay2Location, containerName)
	mergedUrl := containerUrl + "merged"
	containerVolumeUrl := mergedUrl + volumeUrls[1]
	// 卸载 volume
	if _, err := exec.Command("umount", containerVolumeUrl).CombinedOutput(); err != nil {
		log.Errorf("umount volume failed %v", err)
		return err
	}
	// 卸载挂载点
	if _, err := exec.Command("umount", mergedUrl).CombinedOutput(); err != nil {
		log.Errorf("umount merged floder failed %v", err)
		return err
	}
	// 删除 mnt 文件夹
	if err := os.RemoveAll(mergedUrl); err != nil {
		log.Infof("remove merged floder %s error", mergedUrl, err)
	}
	return nil
}

// 删除容器文件夹
func DeleteContainerFolder(containerName string) {
	containerFolder := fmt.Sprintf(Overlay2Location,  containerName)
	if err := os.RemoveAll(containerFolder); err != nil {
		log.Errorf("remove dir %s error %v", containerFolder, err)
	}
}

// 挂载 volume
func MountVolume(volumeUrls []string, containerName string) error {
	parentUrl := volumeUrls[0]
	// 判断宿主机是否存在该文件目录，不存在则创建
	exist, _ := PathExists(parentUrl)
	if !exist {
		// 创建宿主机文件目录
		if err := os.Mkdir(parentUrl, 0777); err != nil {
			log.Errorf("mkdir parent dir %s error %v", parentUrl, err)
		}
	}
	// 在容器文件系统中创建挂载点
	containerUrl := fmt.Sprintf(Overlay2Location, containerName)
	containerVolumeUrl := containerUrl + "merged" + volumeUrls[1]
	if err := os.Mkdir(containerVolumeUrl, 0777); err != nil {
		log.Errorf("mkdir container dir %s error %v", containerVolumeUrl, err)
	}
	// 把宿主机文件目录挂载到容器挂载点
	_, err := exec.Command("mount", "-o", "bind", parentUrl, containerVolumeUrl).CombinedOutput()
	if err != nil {
		log.Errorf("mount volume failed %v", err)
		return err
	}
	return nil
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


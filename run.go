package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"lumper/cgroups/subsystems"
	"os"
	"lumper/container"
	"strconv"
	"strings"
	"lumper/cgroups"
	"math/rand"
	"time"
)

func Run(tty bool, cmdArray []string, res * subsystems.ResourceConfig, containerName string, volume string)  {
	parent, writePipe := container.NewParentProcess(tty, containerName)
	if parent == nil {
		log.Errorf("new parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}

	containerName, err := recordContainerInfo(parent.Process.Pid, cmdArray, containerName)
	if err != nil {
		log.Errorf("record container info error %v", err)
		return
	}
	// 创建 Cgroup Manager
	cgroupManager := cgroups.NewCgroupManager("lumper-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)
	sendInitCommand(cmdArray, writePipe)
	if tty {
		parent.Wait()
		deleteContainerInfo(containerName)
	}
	mntURL := "/root/mnt/"
	rootURL := "/root/"
	container.DeleteWorkSpace(rootURL, mntURL, volume)
}

func sendInitCommand(cmdArray []string, writePipe *os.File)  {
	command := strings.Join(cmdArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

// 记录容器信息
func recordContainerInfo(containerPID int, cmdArray []string, containerName string) (string, error) {
	// 生成 12 位数字容器 ID
	id := randStringBytes(12)
	createTime := time.Now().Format("2006/1/2 15:04:05")
	command := strings.Join(cmdArray, "")
	if containerName == "" {
		containerName = id
	}
	containerInfo := &container.ContainerInfo{
		Pid:         strconv.Itoa(containerPID),
		Id:          id,
		Name:        containerName,
		Command:     command,
		CreatedTime: createTime,
		Status:      container.RUNNING,
	}

	// 将容器信息对象序列号成字符串
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)

	// 拼接容器信息储存路径
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	// 如果路径不存在则创建
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		log.Errorf("mkdir %s error %v", dirUrl, err)
		return "", err
	}
	fileName := dirUrl + "/" + container.ConfigName
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("create file %s error %v", fileName, err)
		return "", err
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("file write string error %v", err)
		return "", err
	}
	return containerName, nil
}

// 删除容器信息
func deleteContainerInfo(containerId string)  {
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	if err := os.RemoveAll(dirUrl); err != nil {
		log.Errorf("remove dir %s error %v", dirUrl, err)
	}
}

// 随机生成 n 位数的字符串
func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
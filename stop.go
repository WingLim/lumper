package main

import (
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"os"
	"strconv"
	"lumper/container"
	"fmt"
	"io/ioutil"
	"encoding/json"
)

func stopContainer(containerName string)  {
	// 根据容器名获取主进程 PID
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		log.Errorf("get container pid by name %s error %v", containerName, err)
		return
	}
	// 将 string 类型的 PID 转换成 int 类型
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		log.Errorf("convert pid from string to int error %v", err)
		return
	}
	// 调用 kill 发送 SIGTERM 信号给进程，从而杀掉容器主进程
	if err := unix.Kill(pidInt, unix.SIGTERM); err != nil {
		log.Errorf("stop container %s error %v", containerName, err)
		return
	}
	// 根据容器配置文件获取信息，并转换成容器信息对象
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		log.Errorf("get container %s info error %v", containerName, err)
		return
	}
	containerInfo.Status = container.STOP
	containerInfo.Pid = " "
	newContentBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("json marshal %s error %v", containerName, err)
		return
	}
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirUrl + container.ConfigName
	if err := ioutil.WriteFile(configFilePath, newContentBytes, 0622); err != nil {
		log.Errorf("write file %s error %v", configFilePath, err)
	}
}

func removeContainer(containerName string)  {
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		log.Errorf("get container %s info error %v", containerName, err)
		return
	}
	if containerInfo.Status != container.STOP {
		log.Errorf("couldn't remove running container")
		return
	}
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirUrl); err != nil {
		log.Errorf("remove file %s error %v", dirUrl, err)
		return
	}
}

func getContainerPidByName(containerName string) (string, error) {
	dirUrl:= fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirUrl + container.ConfigName
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return "", err
	}
	return containerInfo.Pid, nil
}

// 根据容器名获取容器对象
func getContainerInfoByName(containerName string) (*container.ContainerInfo, error) {
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirUrl + container.ConfigName
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Errorf("read file %s error %v", configFilePath, err)
		return nil, err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		log.Errorf("json unmarshal error %v", err)
		return nil, err
	}
	return &containerInfo, nil
}

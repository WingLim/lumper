package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"text/tabwriter"
	"lumper/container"
)

func ListContainers()  {
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, "")
	dirUrl = dirUrl[:len(dirUrl)-1]
	files, err := ioutil.ReadDir(dirUrl)
	if err != nil {
		log.Errorf("read dir %s error %v", dirUrl, err)
		return
	}
	var containers []*container.ContainerInfo
	for _, file := range files {
		// 根据容器配置文件获取信息，并转换成容器信息对象
		tmpContainer, err := getContainerInfo(file)
		if err != nil {
			log.Errorf("get container info error %v", err)
			continue
		}
		containers = append(containers, tmpContainer)
	}
	// 使用 tabwriter 打印容器信息
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreatedTime)
	}
	// 刷新标准输出流缓存区，将容器列表打印出来
	if err := w.Flush(); err != nil {
		log.Errorf("flush container info error %v", err)
		return
	}
}

func getContainerInfo(file os.FileInfo) (*container.ContainerInfo, error) {
	containerName := file.Name()
	configFileDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFileDir = configFileDir + container.ConfigName
	// 读取 config.json 文件中的容器信息
	content, err := ioutil.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("read file %s error %v",configFileDir, err)
		return nil, err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		log.Errorf("json unmarshal error %v", err)
		return nil, err
	}
	return &containerInfo, nil
}

package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"lumper/container"
	"os"
	"io/ioutil"
)

var logCommand = cli.Command{
	Name:   "logs",
	Usage:  "print logs of container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		logContainer(containerName)
		return nil
	},
}

func logContainer(containerName string)  {
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logFileLocation := dirUrl + container.ContainerLogFile
	// 打开日志文件
	file, err := os.Open(logFileLocation)
	defer file.Close()
	if err != nil {
		log.Errorf("open log file %s error %v", logFileLocation, err)
		return
	}
	// 读取日志内容
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("read log file %s error %v",logFileLocation, err)
	}
	// 输出到控制台
	fmt.Fprint(os.Stdout, string(content))
}

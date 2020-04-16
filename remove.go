package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"lumper/container"
	"lumper/network"
	"os"
)

var removeCommand = cli.Command{
	Name:   "remove",
	Aliases: []string{"rm"},
	Usage:  "Remove unused container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		removeContainer(containerName)
		return nil
	},
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
	// 删除时释放网络
	if containerInfo.Network != "" {
		network.Init()
		if err = network.ReleaseContainerNetwork(containerInfo); err != nil {
			log.Errorf("release container network error %v", err)
			return
		}
	}
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirUrl); err != nil {
		log.Errorf("remove file %s error %v", dirUrl, err)
		return
	}
}
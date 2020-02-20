package main

import (
	"fmt"
	"github.com/urfave/cli"
	"lumper/container"
	"os/exec"
	log "github.com/sirupsen/logrus"
)

var commitCommand = cli.Command{
	Name:   "commit",
	Usage:  "commit a container into image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 2 {
			return fmt.Errorf("missing container name or image name")
		}
		containerName := context.Args().Get(0)
		imageName := context.Args().Get(1)
		commitContainer(containerName, imageName)
		return nil
	},
}

func commitContainer(containerName, imageName string)  {
	mntUrl := fmt.Sprintf(container.MntUrl, containerName)
	imageTar := container.RootUrl + imageName + ".tar"
	fmt.Printf("%s", imageTar)
	// 打包容器 rootfs
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntUrl, ".").CombinedOutput(); err != nil {
		log.Errorf("tar folder %s error %v", mntUrl, err)
	}
}

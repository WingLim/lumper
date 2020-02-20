package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"lumper/container"
)

// 初始化容器
var initCommand = cli.Command{
	Name:   "init",
	Usage:  "init container process",
	Action: func(context *cli.Context) error {
		log.Infof("initing")
		err := container.RunContainerInitProcess()
		return err
	},
}
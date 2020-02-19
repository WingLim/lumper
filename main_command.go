package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"lumper/container"
	"lumper/cgroups/subsystems"
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

// 启动一个新容器
var runCommand = cli.Command{
	Name:   "run",
	Usage:  "create a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}
		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}
		tty := context.Bool("t")
		resConf := &subsystems.ResourceConfig{
			MemoryLimit:context.String("m"),
		}
		// 启动容器
		Run(tty, cmdArray, resConf)
		return nil
	},
	Flags:  []cli.Flag{
		cli.BoolFlag{
			Name:  "t",
			Usage: "enable tty",
		},
		cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
		},
	},
}
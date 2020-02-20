package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"lumper/container"
	"lumper/cgroups/subsystems"
	"os"
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
		detach := context.Bool("d")

		// tty 和 detach 不同时执行
		if detach && tty {
			tty = false
		}

		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuShare: context.String("cpushare"),
			CpuSet: context.String("cpuset"),
		}
		containerName := context.String("name")
		// 启动容器
		Run(tty, cmdArray, resConf, containerName)
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
		cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpushare limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
	},
}

var listCommand = cli.Command{
	Name:   "list",
	Usage:  "list all the containers",
	Action: func(context *cli.Context) error {
		ListContainers()
		return nil
	},
}

var stopCommand = cli.Command{
	Name:   "stop",
	Usage:  "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name:   "rm",
	Usage:  "remove unused container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		removeContainer(containerName)
		return nil
	},
}

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

var execCommand = cli.Command{
	Name:   "exec",
	Usage:  "exec command in container",
	Action: func(context *cli.Context) error {
		if os.Getenv(ENV_EXEC_PID) != "" {
			log.Infof("pidf callback pid %s", os.Getpid())
			return nil
		}
		if len(context.Args()) < 2 {
			return fmt.Errorf("mising container name or command")
		}
		containerName := context.Args().Get(0)
		var cmdArray []string
		for _, arg := range context.Args().Tail() {
			cmdArray = append(cmdArray, arg)
		}
		ExecContainer(containerName, cmdArray)
		return nil
	},
}
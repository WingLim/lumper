package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

const usage = "lumper is a simple container runntime implementation"

func main()  {
	app := cli.NewApp()
	app.Name = "lumper"
	app.Version = "0.7"
	app.Usage = usage

	app.Commands = []cli.Command{
		initCommand,
		runCommand,
		listCommand,
		stopCommand,
		removeCommand,
		logCommand,
		execCommand,
		commitCommand,
	}

	app.Before = func(context *cli.Context) error {
		// 以 json 格式输出日志
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}

	if err := app.Run(os.Args); err !=nil {
		log.Fatal(err)
	}

}
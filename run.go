package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"lumper/cgroups/subsystems"
	"os"
	"lumper/container"
	"strconv"
	"strings"
	"lumper/cgroups"
	"math/rand"
	"time"
)

// 启动一个新容器
var runCommand = cli.Command{
	Name:   "run",
	Usage:  "Create a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}
		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}

		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]
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
		volume := context.String("v")
		env := context.StringSlice("e")
		// 启动容器
		Run(tty, cmdArray, env, resConf, containerName, volume, imageName)
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
		cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "set environment",
		},
	},
}

func Run(tty bool, cmdArray, env []string, res * subsystems.ResourceConfig, containerName, volume, imageName string)  {
	containerID := randStringBytes(12)
	if containerName == "" {
		containerName = containerID
	}
	parent, writePipe := container.NewParentProcess(tty, containerName, volume, imageName, env)
	if parent == nil {
		log.Errorf("new parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}

	containerName, err := recordContainerInfo(parent.Process.Pid, cmdArray, containerName, containerID,volume)
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
		container.DeleteWorkSpace(volume, containerName, imageName)
	}
}

func sendInitCommand(cmdArray []string, writePipe *os.File)  {
	command := strings.Join(cmdArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

// 记录容器信息
func recordContainerInfo(containerPID int, cmdArray []string, containerName, id, volume string) (string, error) {
	createTime := time.Now().Format("2006/1/2 15:04:05")
	command := strings.Join(cmdArray, "")
	containerInfo := &container.ContainerInfo{
		Pid:         strconv.Itoa(containerPID),
		Id:          id,
		Name:        containerName,
		Command:     command,
		CreatedTime: createTime,
		Status:      container.RUNNING,
		Volume:      volume,
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
	fileName := dirUrl + container.ConfigName
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
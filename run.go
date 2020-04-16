package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"lumper/cgroups/subsystems"
	"lumper/network"
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
		tty := context.Bool("tty")
		detach := context.Bool("detach")

		// tty 和 detach 不同时执行
		if detach && tty {
			tty = false
		}

		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("memory"),
			CpuShare: context.String("cpushare"),
			CpuSet: context.String("cpuset"),
		}
		containerName := context.String("name")
		volume := context.String("volume")
		env := context.StringSlice("env")
		nw := context.String("net")
		portmapping := context.StringSlice("port")
		// 启动容器
		Run(tty, cmdArray, env, portmapping, resConf, containerName, volume, imageName, nw)
		return nil
	},
	Flags:  []cli.Flag{
		cli.BoolFlag{
			Name:  "tty, t",
			Usage: "enable tty",
		},
		cli.StringFlag{
			Name:  "memory, m",
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpushare, c",
			Usage: "cpushare limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		cli.BoolFlag{
			Name:  "detach, d",
			Usage: "detach container",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
		cli.StringFlag{
			Name:  "volume, v",
			Usage: "volume",
		},
		cli.StringSliceFlag{
			Name:  "env, e",
			Usage: "set environment",
		},
		cli.StringFlag{
			Name:  "net",
			Usage: "container network",
		},
		cli.StringSliceFlag{
			Name: "port, p",
			Usage: "port mapping",
		},
	},
}

func Run(tty bool, cmdArray, env, portmapping []string, res * subsystems.ResourceConfig, containerName, volume, imageName, nw string)  {
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

	createTime := time.Now().Format("2006/1/2 15:04:05")
	command := strings.Join(cmdArray, "")
	containerInfo := &container.ContainerInfo{
		Pid:         strconv.Itoa(parent.Process.Pid),
		Id:          containerID,
		Name:        containerName,
		Command:     command,
		CreatedTime: createTime,
		Status:      container.RUNNING,
		Network:	 nw,
		Volume:      volume,
		PortMapping: portmapping,
	}

	// 创建 Cgroup Manager
	cgroupManager := cgroups.NewCgroupManager("lumper-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)

	if nw != "" {
		network.Init()
		if err := network.Connect(nw, containerInfo); err != nil {
			log.Errorf("connect network error %v", err)
			return
		}
	}

	containerName, err := recordContainerInfo(containerInfo)
	if err != nil {
		log.Errorf("record container info error %v", err)
		return
	}

	sendInitCommand(cmdArray, writePipe)
	if tty {
		parent.Wait()
		deleteContainerInfo(containerName)
		container.DeleteWorkSpace(volume, containerName, imageName)
		if nw != "" {
			network.ReleaseContainerNetwork(containerInfo)
		}
	}
}

func sendInitCommand(cmdArray []string, writePipe *os.File)  {
	command := strings.Join(cmdArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

// 记录容器信息
func recordContainerInfo(cinfo *container.ContainerInfo) (string, error) {
	// 将容器信息对象序列号成字符串
	jsonBytes, err := json.Marshal(cinfo)
	if err != nil {
		log.Errorf("record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)

	// 拼接容器信息储存路径
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, cinfo.Name)
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
	return cinfo.Name, nil
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
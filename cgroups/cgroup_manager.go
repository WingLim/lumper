package cgroups

import (
	"lumper/cgroups/subsystems"
	log "github.com/sirupsen/logrus"
)

type CgroupManager struct {
	// Cgroup 在 hierarchy 中的路径
	Path string
	// 资源配置
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager  {
	return &CgroupManager{
		Path:     path,
	}
}

// 设置各个 Subsystem 挂载中的 Cgroup 资源限制
func (c *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		subSysIns.Set(c.Path, res)
	}
	return nil
}

// 将进程 PID 加入到每个 Cgroup 中
func (c *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		subSysIns.Apply(c.Path, pid)
	}
	return nil
}

// 释放各个 Subsystem 挂载中的 Cgroup
func (c *CgroupManager) Destroy() error {
	for _, SubSysIns := range subsystems.SubsystemsIns {
		if err := SubSysIns.Remove(c.Path); err != nil {
			log.Warnf("remove cgroup fail %v", err)
		}
	}
	return nil
}


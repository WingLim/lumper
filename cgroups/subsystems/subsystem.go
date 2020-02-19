package subsystems

type ResourceConfig struct {
	MemoryLimit string
}

type Subsystem interface {
	// 返回 Subsystem 的名字
	Name() string
	// 设置资源限制
	Set(path string, res *ResourceConfig) error
	// 将进程添加到 cgroup 中
	Apply(path string, pid int) error
	// 移除 cgroup
	Remove(path string) error
}

var (
	SubsystemsIns = []Subsystem{
		&MemorySubSystem{},
	}
)
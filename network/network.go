package network

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"lumper/container"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/lib/lumper/network/network/"
	drivers = map[string]NetworkDriver{}
	networks = map[string]*Network{}
)

// 网络相关配置
type Network struct {
	Name    string     // 网络名
	IPRange *net.IPNet // IP段
	Driver string // 网络驱动名
}

// 网络端点
type Endpoint struct {
	ID string `json:"id"` // ID
	Device netlink.Veth `json:"dev"` // 设备
	IPAddress net.IP `json:"ip"` // IP地址
	MacAddress net.HardwareAddr `json:"mac"` // Mac地址
	PortMapping []string `json:"portmapping"` // 端口映射
	Network *Network // 网络
}

type NetworkDriver interface {
	// 驱动名
	Name() string
	// 创建网络
	Create(subnet , name string) (*Network, error)
	// 删除网络
	Delete(network Network) error
	// 将容器网络端点连接到网络
	Connect(network *Network, endpoint *Endpoint) error
	// 从网络上移除容器网络端点
	Disconnect(network Network, endpoint *Endpoint) error
}

// 将网络配置信息保存到文件
func (nw *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}

	// 网络名为保存的文件名
	nwPath := path.Join(dumpPath, nw.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC | os.O_WRONLY | os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("error: %v", err)
		return err
	}
	defer nwFile.Close()

	// 序列化网络对象为 json 字符串
	nwJson, err := json.Marshal(nw)
	if err != nil {
		log.Errorf("error: %v", err)
		return err
	}

	// 将网络配置写入文件
	_, err = nwFile.Write(nwJson)
	if err != nil {
		log.Errorf("error: %v", err)
		return err
	}
	return nil
}

// 从配置文件中加载网络配置
func (nw *Network) load(dumpPath string) error {
	nwConfigFile, err := os.Open(dumpPath)
	defer nwConfigFile.Close()
	if err != nil {
		return err
	}
	// 读取 json
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		log.Errorf("load network info error", err)
		return err
	}
	return nil
}

// 删除配置文件
func (nw *Network) remove(dumpPath string) error {
	if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		return os.Remove(path.Join(dumpPath, nw.Name))
	}
}

// 初始化
func Init() error {
	// 加载网络驱动
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	// 创建网络配置目录
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}

	// 遍历网络配置目录中的文件
	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		// 文件名为网络名
		_, nwName := path.Split(nwPath)
		nw := &Network{Name:nwName}

		if err := nw.load(nwPath); err != nil {
			log.Errorf("load network %s error %v", nwName,err)
		}
		
		networks[nwName] = nw
		return nil
	})
	return nil
}

// 创建网络
func CreateNetwork(driver, subnet, name string) error {
	// 将网段字符串转换成 net.IPNet 对象
	_, cidr, _ := net.ParseCIDR(subnet)
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = ip
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err !=nil {
		return err
	}
	return nw.dump(defaultNetworkPath)
}

// 列出所有网络
func ListNetwork()  {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIPRagne\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n", nw.Name, nw.IPRange.String(), nw.Driver,)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("flush error %v", err)
		return
	}
}

// 删除网络
func DeleteNetwork(networkName string) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network %s", networkName)
	}

	if err := ipAllocator.Release(nw.IPRange, &nw.IPRange.IP); err != nil {
		return fmt.Errorf("remove network ip errorr %v", err)
	}
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("remove network driver error %v", err)
	}
	return nw.remove(defaultNetworkPath)
}

// 释放容器 IP 和移除端口映射
func ReleaseContainerNetwork(cinfo *container.ContainerInfo) error {
	nw, ok := networks[cinfo.Network]
	if !ok {
		return fmt.Errorf("no such network %s", cinfo.Network)
	}
	// 释放 IP
	ip := net.ParseIP(cinfo.IPAddress)
	if err := ipAllocator.Release(nw.IPRange, &ip); err != nil {
		return fmt.Errorf("remove network ip errorr %v", err)
	}
	// 移除端口映射
	if err := removePortMapping(cinfo); err != nil {
		log.Errorf("remove portmapping error %v", err)
		return err
	}
	return nil
}

// 进入容器的 Net Namespace
func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("get container net namespace error %v", err)
	}
	// 获取文件的文件描述符
	nsFD := f.Fd()
	// 锁定线程，避免被调用到其他线程上
	runtime.LockOSThread()
	// 修改网络端点的另一端到容器的 Net Namespace 中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("set link netns error %v", err)
	}
	// 获取当前网络的 Net Namespace
	origns, err := netns.Get()
	if err != nil {
		log.Errorf("get current netns error %v", err)
	}
	// 将当前进程加入容器的 Net Namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("set netns error %v", err)
	}

	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

// 配置容器网络端的地址和路由
func configNetwork(ep *Endpoint, cinfo *container.ContainerInfo) error {
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("config endpoint failed %v", err)
	}
	defer enterContainerNetns(&peerLink, cinfo)()

	interfaceIP := *ep.Network.IPRange
	interfaceIP.IP = ep.IPAddress
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%s error %v", ep.Network, err)
	}

	// 启动容器内的 Veth 端点
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}

	// 启动容器内的 lo 网卡
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	// 设置容器内的外部请求都通过 Veth 端点访问
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Dst:       cidr,
		Gw:        ep.Network.IPRange.IP,
	}

	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			log.Errorf("port mapping format error %v", pm)
			continue
		}
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("iptables error %v", output)
			continue
		}
	}
	return nil
}

func removePortMapping(cinfo *container.ContainerInfo) error {
	for _, pm := range cinfo.PortMapping {
		portMapping := strings.Split(pm, ":")
		iptablesCmd := fmt.Sprintf("-t nat -D PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", portMapping[0], cinfo.IPAddress, portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("iptables error %v", output)
			continue
		}
	}
	return nil
}

// 连接网络
func Connect(networkName string, cinfo *container.ContainerInfo) error {
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network %s", networkName)
	}

	// 从 IP 段中分配容器 IP 地址
	ip, err := ipAllocator.Allocate(network.IPRange)
	if err != nil {
		return err
	}

	// 创建网络端点并设置
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, networkName),
		IPAddress:   ip,
		PortMapping: cinfo.PortMapping,
		Network:     network,
	}

	// 保存容器 IP 地址
	cinfo.IPAddress = ip.String()

	// 挂载网络
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}

	// 配置网络
	if err = configNetwork(ep, cinfo); err != nil {
		return err
	}

	// 配置端口映射
	if err = configPortMapping(ep, cinfo); err != nil {
		return err
	}

	return nil
}

func Disconnect(networkName string, cinfo *container.ContainerInfo) error {
	return nil
}
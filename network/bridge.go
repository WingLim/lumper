package network

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
)

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

// 创建网络
func (d *BridgeNetworkDriver) Create(subnet, name string) (*Network, error) {
	// 获取网关 IP 地址和网络 IP 段
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
	// 初始化网络对象
	n := & Network{
		Name:    name,
		IPRange: ipRange,
		Driver:  d.Name(),
	}
	err := d.initBridge(n)
	if err != nil {
		log.Errorf("init bridge error %v", err)
	}
	return  n, err
}

// 删除网络
func (d *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}

// 连接网络和网络端点
func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	
	la := netlink.NewLinkAttrs()
	// 取 ID 前五位
	la.Name = endpoint.ID[:5]
	la.MasterIndex = br.Attrs().Index

	endpoint.Device = netlink.Veth{
		LinkAttrs:        la,
		PeerName:         "cif-" + endpoint.ID[:5],
	}
	
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("add endpoint device error %v", err)
	}
	
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("set endpoing device error %v", err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

// 初始化 Bridge 设备
func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("add bridge %s error %v", bridgeName, err)
	}
	gatewayIP := *n.IPRange
	gatewayIP.IP = n.IPRange.IP
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("assign address %s on bridge %s error %v", gatewayIP, bridgeName, err)
	}
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("set bridge %s up error %v", bridgeName, err)
	}
	if err := setupIPTables(bridgeName, n.IPRange); err != nil {
		return fmt.Errorf("set iptables for %s error %v", bridgeName, err)
	}
	return nil
}

// 创建 Bridge 设备
func createBridgeInterface(bridgeName string) error {
	// 判断 Bridge 是否存在
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	
	br := &netlink.Bridge{LinkAttrs: la}
	if err := netlink.LinkAdd(br); err !=nil {
		return fmt.Errorf("create bridge %s failed %v", bridgeName, err)
	}
	return nil
}

// 设置网络接口 IP 地址
func setInterfaceIP(name, rawIP string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("get interface error %v", err)
	}
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	addr := &netlink.Addr{ipNet, "", 0, 0, nil, nil, 0, 0}
	return netlink.AddrAdd(iface, addr)
}

// 启动 Bridge 设备
func setInterfaceUP(interfaceName string) error {
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("retrieving a link named %s error %v", interfaceName, err)
	}
	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("enabling interface for %s error %v", interfaceName, err)
	}
	return nil
}

// 设置 Bridge 的 MASQUERADE 规则
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables error %v", output)
	}
	return nil
}
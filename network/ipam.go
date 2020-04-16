package network

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"path"
	"strings"
)

// 默认分配文件路径
const ipamDefaultAllocatorPath = "/var/lib/lumper/network/ipam/subnet.json"

type IPAM struct {
	// 分配文件存放路径
	SubnetAllocatorPath string
	// 网段和位图算法的的数组，网段为 key，分配的位图数组为 value
	Subnets *map[string]string
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// 加载网段地址分配信息
func (ipam *IPAM) load() error {
	// 检查文件状态，如果不存在则没有分配，不需要加载
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err !=nil {
		if os.IsNotExist(err){
			return nil
		} else {
			return err
		}
	}
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}
	// 通过反序列化获取 IP 分配信息
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		log.Errorf("dump allocation info error %v", err)
		return err
	}
	return nil
}

// 储存网段地址分配信息
func (ipam *IPAM) dump() error {
	// 检测储存配置文件的文件夹是否存在，不存在则创建
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(ipamConfigFileDir, 0644)
		} else {
			return err
		}
	}
	// 打开配置文件，O_TRUNC：存在则清空，O_WRONLY：以只写方式打开，O_CREATE：不存在则创建
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC | os.O_WRONLY | os.O_CREATE, 0644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	// 将 ipam 对象序列化为 json
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	// 将 json 写入到配置文件中
	if _, err = subnetConfigFile.Write(ipamConfigJson); err != nil {
		return err
	}
	return nil
}

// 在网段中分配一个可用的 IP 地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	ipam.Subnets = &map[string]string{}
	// 加载已分配的网段信息
	err = ipam.load()
	if err != nil {
		log.Errorf("load allocation info error %v", err)
	}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	one, size := subnet.Mask.Size()

	// 如果网段没有分配过，则初始化该网段
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1 << uint8(size - one))
	}

	// 遍历网段位图数组
	for c := range((*ipam.Subnets)[subnet.String()]) {
		// 找到数组中为 "0" 的项和数组序号，即可分配的 IP
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			// 将 "0" 设置为 "1" ，分配这个 IP
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			ip = subnet.IP
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4 - t] += uint8(c >> ((t - 1) * 8))
			}
			// IP 是从 1 开始分配，所以这里加 1
			ip[3] += 1
			break
		}
	}
	// 将分配结果保存到文件中
	ipam.dump()
	return
}

// 释放 IP
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	// 从文件中加载网段分配信息
	err := ipam.load()
	if err != nil {
		log.Errorf("dump allocation info error %v", err)
	}
	// 计算 IP 地址在网段位图数组中的索引位置
	c := 0
	// 将 IP 地址转换成 4 个字节的表示方式
	releaseIP := ipaddr.To4()
	// IP 是从 1 开始分配，转换成引索减 1
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t - 1] - subnet.IP[t - 1]) << ((4 - t) * 8)
	}

	// 将分配的位图数组中引索位置置零
	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)
	// 保存释放掉 IP 后的网段 IP 分配信息
	ipam.dump()
	return nil
}
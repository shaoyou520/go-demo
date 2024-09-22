package main

import (
	"fmt"
	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
)

/*
#实验环境#
主机 A:
HOST: 192.168.0.3
ns0: 10.0.1.1/24
bridge0: 10.0.1.254/24
主机 B:
HOST: 192.168.0.2
ns0: 10.0.2.1/24
bridge0: 10.0.2.254/24
#实现目标#
10.0.1.1 -$ 192.168.0.2
10.0.1.1 -$ 10.0.2.1
*/

var (
	IsHostA = false
)

const (
	EnvName = "IS_HOST_A"
	// 无需修改
	NSName     = "ns0"
	BridgeName = "bridge0"
	VEth0      = "veth0"
	NsEth0     = "nseth0"

	// 如果需要在自己的机器上实现，需要改动 HostA 和 HostB 变量即可
	HostA        = "192.168.0.3/24"
	HostANS0     = "10.0.1.1/24"
	HostABridge0 = "10.0.1.254/24"

	HostB        = "192.168.0.2/24"
	HostBNS0     = "10.0.2.1/24"
	HostBBridge0 = "10.0.2.254/24"
)

func Error(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

type DoFunc func(oNs, nNs *netns.NsHandle) error

func SetupBridge() *netlink.Bridge {
	log.Println("SetupBridge...runing")
	linkDev, _ := netlink.LinkByName(BridgeName)

	if linkDev != nil {
		if _, ok := linkDev.(*netlink.Bridge); ok {
			Error(netlink.LinkDel(linkDev))
		}
	}

	br0 := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:   BridgeName,
			MTU:    1500,
			TxQLen: -1,
		},
	}
	// 添加
	Error(netlink.LinkAdd(br0))
	// 启动
	Error(netlink.LinkSetUp(br0))
	// 设置 ip
	hb := HostABridge0
	if !IsHostA {
		hb = HostBBridge0
	}

	ipv4Net, err := netlink.ParseIPNet(hb)
	Error(err)
	Error(netlink.AddrAdd(br0, &netlink.Addr{
		IPNet: ipv4Net,
	}))
	log.Println("SetupBridge...done")
	return br0
}

func SetupNetNamespace() *netns.NsHandle {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	log.Println("SetupNetNamespace...running")
	origns, _ := netns.Get()
	defer origns.Close()
	_, err := netns.GetFromName(NSName)
	Error(netns.Set(origns))

	if err == nil {
		log.Printf("%s net ns is exists. Delete netns %s\n", NSName, NSName)
		cmd := exec.Command(
			"sh",
			"-c",
			fmt.Sprintf("/usr/sbin/ip netns del %s", NSName))
		Error(cmd.Run())
	}

	// 由于程序上启动 netns 是需要附着于进程上的，所以这里直接使用 ip netns 来创建 net namespace
	cmd := exec.Command("sh", "-c", fmt.Sprintf("/usr/sbin/ip netns add %s", NSName))
	Error(cmd.Run())
	ns, err := netns.GetFromName(NSName)
	Error(err)
	log.Println("SetupNetNamespace...done")
	return &ns
}

func SetupVEthPeer(br *netlink.Bridge, ns *netns.NsHandle) {
	log.Println("SetupVEth...running")

	oBrVEth, _ := netlink.LinkByName(VEth0)
	if oBrVEth != nil {
		log.Printf("%s is exists, it will be delete. \n", VEth0)
		Error(netlink.LinkDel(oBrVEth))
	}

	log.Printf("Create vethpeer %s peer name %s\n", VEth0, NsEth0)
	vethPeer := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: VEth0,
		},
		PeerName: NsEth0,
	}

	Error(netlink.LinkAdd(vethPeer))

	log.Printf("Set %s to %s \n", vethPeer.PeerName, NSName)
	// 获取 ns 的 veth
	nsVeth, err := netlink.LinkByName(vethPeer.PeerName)
	Error(err)
	// 设置 ns 的 veth
	Error(netlink.LinkSetNsFd(nsVeth, int(*ns)))

	log.Printf("Set %s master to %s \n", vethPeer.Name, BridgeName)
	// 获取 host 的 veth
	brVeth, err := netlink.LinkByName(vethPeer.Name)
	Error(err)
	// 设置 bridge 的 veth
	Error(netlink.LinkSetMaster(brVeth, br))
	log.Printf("Set up %s \n", vethPeer.Name)
	Error(netlink.LinkSetUp(brVeth))

	Error(NsDo(func(oNs, nNs *netns.NsHandle) error {
		// 设置 IP 地址
		hn := HostANS0
		if !IsHostA {
			hn = HostBNS0
		}
		log.Printf("Addr add ip to %s \n", vethPeer.Name)
		ipv4Net, err := netlink.ParseIPNet(hn)
		Error(err)
		Error(netlink.AddrAdd(nsVeth, &netlink.Addr{
			IPNet: ipv4Net,
		}))
		log.Printf("Set up %s \n", vethPeer.PeerName)
		// 启动设备
		Error(netlink.LinkSetUp(nsVeth))

		log.Println("SetupVEth...done")
		return nil
	}))

}

func SetupNsDefaultRoute() {
	Error(NsDo(func(oNs, nNs *netns.NsHandle) error {

		// add default gate way
		log.Println("SetupNsDefaultRouter... running")
		log.Printf("Add net namespace %s default gateway.", NSName)

		gwAddr := HostBBridge0
		if IsHostA {
			gwAddr = HostABridge0
		}

		gw, err := netlink.ParseIPNet(gwAddr)
		Error(err)

		defaultRoute := &netlink.Route{
			Dst: nil,
			Gw:  gw.IP,
		}

		Error(netlink.RouteAdd(defaultRoute))
		log.Println("SetupNsDefaultRouter... done")
		return nil
	}))
}

func SetupIPTables() {
	log.Println("Setup IPTables for ns transfer packet to remote host...running")
	// 读取本地 iptables，默认是 ipv4
	ipt, err := iptables.New()
	Error(err)
	//  iptables -t nat -A POSTROUTING -s 10.0.1.0/24 -j MASQUERADE
	nsAddr := HostBNS0
	if IsHostA {
		nsAddr = HostANS0
	}
	_, srcNet, err := net.ParseCIDR(nsAddr)
	Error(err)

	rule := []string{"-s", srcNet.String(), "-j", "MASQUERADE"}
	// 先进行清除后再添加，保持简易幂等
	_ = ipt.Delete("nat", "POSTROUTING", rule...)
	Error(ipt.Append("nat", "POSTROUTING", rule...))
	log.Println("Setup IPTables done")
}

func SetupRouteNs2Ns() {
	log.Println("Setup route HostA(net0) <==$ HostB(net0)... running")
	//  10.0.1.0/24 via 192.168.0.3 dev eth0
	gwAddr := HostA
	dstAddr := HostANS0
	if IsHostA {
		gwAddr = HostB
		dstAddr = HostBNS0
	}

	_, dstNet, err := net.ParseCIDR(dstAddr)
	Error(err)

	gwNet, err := netlink.ParseIPNet(gwAddr)
	Error(err)

	// 先删除后增加
	_ = netlink.RouteDel(&netlink.Route{
		LinkIndex: netlink.NewLinkAttrs().Index,
		Dst:       dstNet,
	})

	Error(netlink.RouteAdd(&netlink.Route{
		LinkIndex: netlink.NewLinkAttrs().Index,
		Dst:       dstNet,
		Gw:        gwNet.IP,
	}))

	log.Println("Setup route HostA(net0) <==$ HostB(net0) done")
}

func NsDo(doFunc DoFunc) error {
	// 进入 netns, 使用线程锁固定当前 routine 不会其他线程执行，避免 namespace 执行出现异常。
	// 因为如果在执行过程中切换了线程，可能找不到已经建立好的 namespace。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Printf("Switch net namespace to %s \n", NSName)
	originNs, err := netns.Get()
	Error(err)
	// 切换回原始 ns
	defer func() {
		log.Println("Switch net namespace to origin")
		Error(netns.Set(originNs))
	}()

	ns, err := netns.GetFromName(NSName)
	Error(err)

	Error(netns.Set(ns))
	return doFunc(&originNs, &ns)
}

func main() {

	if os.Getenv(EnvName) == "1" {
		IsHostA = true
	}
	ns := SetupNetNamespace()
	bridge := SetupBridge()
	SetupVEthPeer(bridge, ns)
	SetupNsDefaultRoute()
	SetupIPTables()
	SetupRouteNs2Ns()
	log.Println("Config Finished.")
}

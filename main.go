package main

import (
	bcm "bcm-eth-switch-mgmt/switch"
	"fmt"
	"net"
	"os"
	"os/signal"

	"github.com/beluganos/go-opennsl/examples/util"
	"github.com/beluganos/go-opennsl/opennsl"
	"github.com/beluganos/go-opennsl/sal"
	log "github.com/sirupsen/logrus"
)

func watchSignal(done chan struct{}) {
	ch := make(chan os.Signal, 1)
	defer close(ch)
	signal.Notify(ch, os.Interrupt)
	<-ch
	log.Infof("Interrupt signal.")
	close(done)
}

func main() {
	log.SetLevel(log.DebugLevel)
	sw := bcm.NewSwitch()
	if err := sw.Init(); err != nil {
		log.Errorf("Failed to initialize BCM network switch layer")
		return
	}

	defer sw.Release()

	if err := sw.EnableFeatures(); err != nil {
		log.Errorf("Failed to enable BCM network switch features: %s", err)
		return
	}

	mgmtIface := bcm.NewMgmtIface(
		"cpu-0",
		opennsl.VLAN_ID_DEFAULT,
		util.ParseMAC("00:11:22:33:44:00"),
		net.ParseIP("10.1.1.4"),
	)

	if err := mgmtIface.Create(); err != nil {
		log.Errorf("Failed to create BCM network switch management interface: %s", err)
		return
	}

	l2Ports := make(map[string]*bcm.L2Port)
	var idx uint16 = 0
	// for _, namePortMap := range bcm.NamePortMap {
	// 	fmt.Println("Port name:", namePortMap.PortName, "BCM Port:", namePortMap.Port)
	// 	portMac := fmt.Sprintf("00:11:22:33:%02x:fe", idx)
	// 	macAddr := util.ParseMAC(portMac)
	// 	l2Port := bcm.NewL2Port(namePortMap.PortName, namePortMap.Port, opennsl.VLAN_ID_DEFAULT, macAddr)
	// 	if err := l2Port.Create(int(idx + 1)); err != nil {
	// 		log.Errorf("Failed to create L2 port: %s", err)
	// 		return
	// 	}

	// 	l2Ports[namePortMap.PortName] = l2Port
	// 	idx++
	// }
	for _, portNameMap := range bcm.PortNames {
		fmt.Println("Key:", portNameMap.Port, "Value:", portNameMap.PortName)
		portMac := fmt.Sprintf("00:11:22:33:%02x:fe", idx)
		macAddr := util.ParseMAC(portMac)
		l2Port := bcm.NewL2Port(portNameMap.PortName, portNameMap.Port, opennsl.VLAN_ID_NONE, macAddr)
		if err := l2Port.Create(int(idx + 1)); err != nil {
			log.Errorf("Failed to create L2 port: %s", err)
			return
		}

		l2Ports[portNameMap.PortName] = l2Port
		idx++
	}

	rx := bcm.NewRx()
	if err := rx.Start(); err != nil {
		log.Errorf("Failed to active receiving data: %s", err)
		return
	}

	defer rx.Stop()
	go bcm.HandleSTPRequest(sw)
	go bcm.HandleLAGRequest(sw)

	if err := sal.DriverShell(); err != nil {
		log.Errorf("Failed to exit from driver shell: %s", err)
		return
	}

	done := make(chan struct{})
	go watchSignal(done)
	<-done

	// stp := make(chan struct{})
	// go handleSTPRequest(sw)

	// for {
	// 	select {
	// 	case <-done:
	// 		log.Infoln("Finishing program")
	// 		return
	// 	case <-stp:
	// 		log.Infoln("Get STP request")
	// 		return
	// 	}
	// }
}

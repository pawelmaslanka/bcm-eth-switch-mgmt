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
	var activeSwitchingLayer3 bool = true
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

	if err := sal.DriverShell(); err != nil {
		log.Errorf("Failed to exit from driver shell: %s", err)
		return
	}

	if !activeSwitchingLayer3 {
		mgmtIntf := bcm.NewMgmtIntf(
			"cpu-0",
			opennsl.VLAN_ID_DEFAULT,
			util.ParseMAC("00:11:22:33:44:00"),
			net.ParseIP("10.1.1.4"),
		)

		if err := mgmtIntf.Create(); err != nil {
			log.Errorf("Failed to create BCM network switch management interface: %s", err)
			return
		}

		l2Ports := make(map[string]*bcm.L2Port)
		var idx uint16 = 0
		for _, portNameMap := range bcm.PortNames {
			fmt.Println("Key:", portNameMap.Port, "Value:", portNameMap.PortName)
			portMac := fmt.Sprintf("00:11:22:33:%02x:fe", idx)
			macAddr := util.ParseMAC(portMac)
			l2Port := bcm.NewL2Port(portNameMap.PortName, portNameMap.Port, opennsl.VLAN_ID_NONE, macAddr)
			if err := l2Port.Create(); err != nil {
				log.Errorf("Failed to create L2 port: %s", err)
				return
			}

			l2Ports[portNameMap.PortName] = l2Port
			idx++
		}
	} else {
		l3Ports := make(map[string]*bcm.L3Port)
		var idx uint16 = 1
		for _, portNameMap := range bcm.PortNames {
			fmt.Println("Key:", portNameMap.Port, "Value:", portNameMap.PortName)
			portMac := fmt.Sprintf("00:11:22:33:%02x:fe", idx)
			macAddr := util.ParseMAC(portMac)
			ipAddr := net.ParseIP(fmt.Sprintf("192.168.%d.1", idx))
			l3Port := bcm.NewL3Port(portNameMap.PortName, portNameMap.Port, macAddr, ipAddr)
			if err := l3Port.Create(); err != nil {
				log.Errorf("Failed to create L3 interface: %s", err)
				return
			}

			l3Ports[portNameMap.PortName] = l3Port
			idx++
		}
	}

	rx := bcm.NewRx()
	if err := rx.Start(); err != nil {
		log.Errorf("Failed to active receiving data: %s", err)
		return
	}

	defer rx.Stop()
	go bcm.HandleSTPRequest(sw)
	go bcm.HandleLAGRequest(sw)
	go bcm.HandleVlanMgmtRequest(sw)
	go bcm.HandleRequestOfL3RouteMgmtRpc(sw)

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

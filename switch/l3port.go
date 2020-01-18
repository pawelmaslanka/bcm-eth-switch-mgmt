package bcm

import (
	"fmt"
	"net"

	"github.com/beluganos/go-opennsl/opennsl"
)

type L3Port struct {
	name          string
	asic          Asic
	port          opennsl.Port
	macAddr       net.HardwareAddr
	ipAddr        net.IP
	knetIntfID    opennsl.KnetNetIfaceID
	knetFilterIDs map[string]opennsl.KnetFilterID
	l3IntfID      opennsl.L3IfaceID
	l3EgressID    opennsl.L3EgressID
}

func NewL3Port(name string, port opennsl.Port, macAddr net.HardwareAddr, ipAddr net.IP) *L3Port {
	return &L3Port{
		name:          name,
		asic:          Asic{unit: DEFAULT_ASIC_UNIT},
		port:          port,
		macAddr:       macAddr,
		ipAddr:        ipAddr,
		knetFilterIDs: make(map[string]opennsl.KnetFilterID),
	}
}

func (l3Port *L3Port) setupKnetIntf() error {
	knetIntf := opennsl.NewKnetNetIface()
	knetIntf.SetType(opennsl.KNET_NETIF_T_TX_LOCAL_PORT)
	knetIntf.SetName(l3Port.name)
	knetIntf.SetPort(l3Port.port)
	knetIntf.SetMAC(l3Port.macAddr)
	if err := knetIntf.Create(l3Port.asic.unit); err != nil {
		return err
	}

	l3Port.knetIntfID = knetIntf.ID()
	return nil
}

func (l3Port *L3Port) setupKnetFilter(rxReason opennsl.RxReason, prio opennsl.KnetFilterPrio) error {
	strFilterID := fmt.Sprintf("%s-%s", l3Port.name, rxReason.String())
	knetFilter := opennsl.NewKnetFilter()
	knetFilter.SetDescription(strFilterID)
	knetFilter.SetType(opennsl.KNET_FILTER_T_RX_PKT)
	knetFilter.SetFlags(opennsl.NewKnetFilterFlags(
		opennsl.KNET_FILTER_F_STRIP_TAG,
	))
	knetFilter.SetMatchFlags(opennsl.NewKnetFilterMatchFlags(
		opennsl.KNET_FILTER_M_REASON,
		opennsl.KNET_FILTER_M_INGPORT,
	))
	knetFilter.SetDestType(opennsl.KNET_DEST_T_NETIF)
	knetFilter.SetDestID(l3Port.knetIntfID)
	knetFilter.SetPriority(prio)
	knetFilter.SetRxReason(rxReason)
	knetFilter.SetIngPort(l3Port.port)
	if err := knetFilter.Create(l3Port.asic.unit); err != nil {
		return err
	}

	l3Port.knetFilterIDs[strFilterID] = knetFilter.ID()
	return nil
}

func (l3Port *L3Port) setReplacementForSourceFrameInfo() error {
	l3Intf := opennsl.NewL3Iface()
	l3Intf.SetFlags(opennsl.NewL3Flags(
		opennsl.L3_ADD_TO_ARL,
	))
	l3Intf.SetVID(opennsl.VLAN_ID_DEFAULT)
	l3Intf.SetMAC(l3Port.macAddr)
	if err := l3Intf.Create(l3Port.asic.unit); err != nil {
		return err
	}

	l3Port.l3IntfID = l3Intf.IfaceID()
	return nil
}

func (l3Port *L3Port) addNextHopEntry() error {
	l3eg := opennsl.NewL3Egress()
	l3eg.SetIfaceID(l3Port.l3IntfID)
	l3Flags := opennsl.NewL3Flags(
		opennsl.L3_COPY_TO_CPU,
		opennsl.L3_L2TOCPU,
	)
	l3eg.SetFlags(l3Flags)

	var l3egID opennsl.L3EgressID
	l3EgressID, err := l3eg.Create(l3Port.asic.unit, opennsl.L3_NONE, l3egID)
	if err != nil {
		return err
	}

	l3Port.l3EgressID = l3EgressID
	return nil
}

func (l3Port *L3Port) addHostTableEntry() error {
	l3Host := opennsl.NewL3Host()
	l3Host.SetIPAddr(l3Port.ipAddr)
	l3Host.SetEgressID(l3Port.l3EgressID)

	if err := l3Host.Add(l3Port.asic.unit); err != nil {
		return err
	}

	return nil
}

func (l3Port *L3Port) Create() error {
	var err error
	if err = l3Port.setupKnetIntf(); err != nil {
		return err
	}

	if err = l3Port.setupKnetFilter(opennsl.RxReasonNhop, opennsl.KNET_FILTER_PRIO_NHOP); err != nil {
		return err
	}

	if err = l3Port.setupKnetFilter(opennsl.RxReasonProtocol, opennsl.KNET_FILTER_PRIO_PROTOCOL); err != nil {
		return err
	}

	err = opennsl.PortFloodBlockSet(l3Port.asic.unit, l3Port.port, opennsl.Port(0), opennsl.PORT_FLOOD_BLOCK_UNKNOWN_UCAST)
	if err != nil {
		return err
	}

	if err = opennsl.SwitchArpRequestToCpu.PortSet(l3Port.asic.unit, l3Port.port, opennsl.TRUE); err != nil {
		return err
	}

	if err = opennsl.SwitchArpReplyToCpu.PortSet(l3Port.asic.unit, l3Port.port, opennsl.TRUE); err != nil {
		return err
	}

	if err = opennsl.SwitchL3SlowpathToCpu.PortSet(l3Port.asic.unit, l3Port.port, opennsl.TRUE); err != nil {
		return err
	}

	if err = l3Port.setReplacementForSourceFrameInfo(); err != nil {
		return err
	}

	if err = l3Port.addNextHopEntry(); err != nil {
		return err
	}

	if err = l3Port.addHostTableEntry(); err != nil {
		return err
	}

	return nil
}

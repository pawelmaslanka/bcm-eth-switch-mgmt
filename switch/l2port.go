package bcm

import (
	"net"

	"github.com/beluganos/go-opennsl/opennsl"
)

type l2PortKnetFiltersType map[string]opennsl.KnetFilterID

type L2Port struct {
	asic           Asic
	portName       string
	port           opennsl.Port
	vlan           opennsl.Vlan
	macAddr        net.HardwareAddr
	knetIntfID     opennsl.KnetNetIfaceID
	knetFilterIDs    map[string]opennsl.KnetFilterID
}

func NewL2Port(portName string, port opennsl.Port, vlan opennsl.Vlan, macAddr net.HardwareAddr) *L2Port {
	return &L2Port{
		asic:        Asic{unit: DEFAULT_ASIC_UNIT},
		portName:    portName,
		port:        port,
		vlan:        vlan,
		macAddr:     macAddr,
		knetFilterIDs: make(map[string]opennsl.KnetFilterID),
	}
}

func (l2Port *L2Port) setupKnetNetIntf() error {
	knetNetIntf := opennsl.NewKnetNetIface()
	knetNetIntf.SetType(opennsl.KNET_NETIF_T_TX_LOCAL_PORT)
	knetNetIntf.SetVlan(l2Port.vlan)
	knetNetIntf.SetPort(l2Port.port)
	knetNetIntf.SetMAC(l2Port.macAddr)
	knetNetIntf.SetName(l2Port.portName)
	if err := knetNetIntf.Create(l2Port.asic.unit); err != nil {
		return err
	}

	l2Port.knetIntfID = knetNetIntf.ID()
	return nil
}

func (l2Port *L2Port) setupKnetFilter(rxReason opennsl.RxReason, prio opennsl.KnetFilterPrio, desc string) error {
	knetFilter := opennsl.NewKnetFilter()
	knetFilter.SetDescription(desc)
	knetFilter.SetType(opennsl.KNET_FILTER_T_RX_PKT)
	knetFilter.SetFlags(opennsl.NewKnetFilterFlags(
		opennsl.KNET_FILTER_F_STRIP_TAG,
	))
	knetFilter.SetMatchFlags(opennsl.NewKnetFilterMatchFlags(
		opennsl.KNET_FILTER_M_REASON,
		opennsl.KNET_FILTER_M_INGPORT,
	))
	knetFilter.SetDestType(opennsl.KNET_DEST_T_NETIF)
	knetFilter.SetDestID(l2Port.knetIntfID)
	knetFilter.SetPriority(prio)
	knetFilter.SetRxReason(rxReason)
	knetFilter.SetIngPort(l2Port.port)
	if err := knetFilter.Create(l2Port.asic.unit); err != nil {
		return err
	}

	l2Port.knetFilterIDs[desc] = knetFilter.ID()
	return nil
}

func (l2Port *L2Port) Create() error {
	var err error
	if err = l2Port.setupKnetNetIntf(); err != nil {
		return err
	}

	if err = l2Port.setupKnetFilter(opennsl.RxReasonBpdu, opennsl.KNET_FILTER_PRIO_BPDU, "Catch BPDU"); err != nil {
		return err
	}

	err = opennsl.PortFloodBlockSet(l2Port.asic.unit, l2Port.port, opennsl.Port(0), opennsl.PORT_FLOOD_BLOCK_UNKNOWN_UCAST)
	if err != nil {
		return err
	}

	if err = opennsl.SwitchArpRequestToCpu.PortSet(l2Port.asic.unit, l2Port.port, opennsl.TRUE); err != nil {
		return err
	}

	if err = opennsl.SwitchArpReplyToCpu.PortSet(l2Port.asic.unit, l2Port.port, opennsl.TRUE); err != nil {
		return err
	}

	if err = opennsl.SwitchL3SlowpathToCpu.PortSet(l2Port.asic.unit, l2Port.port, opennsl.TRUE); err != nil {
		return err
	}

	return nil
}

func (l2Port *L2Port) Port() opennsl.Port {
	return l2Port.port
}

func (l2Port *L2Port) KnetIntfID() opennsl.KnetNetIfaceID {
	return l2Port.knetIntfID
}

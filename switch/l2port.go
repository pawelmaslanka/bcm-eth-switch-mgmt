package bcm

import (
	"net"

	"github.com/beluganos/go-opennsl/opennsl"
)

type l2PortKnetFiltersType map[string]int

type L2Port struct {
	asic           Asic
	portName       string
	port           opennsl.Port
	vlan           opennsl.Vlan
	macAddr        net.HardwareAddr
	knetNetIfaceID int
	knetFilters    l2PortKnetFiltersType
}

func NewL2Port(portName string, port opennsl.Port, vlan opennsl.Vlan, macAddr net.HardwareAddr) *L2Port {
	return &L2Port{
		asic:        Asic{unit: DEFAULT_ASIC_UNIT},
		portName:    portName,
		port:        port,
		vlan:        vlan,
		macAddr:     macAddr,
		knetFilters: make(l2PortKnetFiltersType),
	}
}

func (l2Port *L2Port) setupKnetNetIface() error {
	knetNetIface := opennsl.NewKnetNetIface()
	knetNetIface.SetType(opennsl.KNET_NETIF_T_TX_LOCAL_PORT)
	knetNetIface.SetVlan(l2Port.vlan)
	knetNetIface.SetPort(l2Port.port)
	knetNetIface.SetMAC(l2Port.macAddr)
	knetNetIface.SetName(l2Port.portName)
	if err := knetNetIface.Create(l2Port.asic.unit); err != nil {
		return err
	}

	l2Port.knetNetIfaceID = knetNetIface.ID()
	return nil
}

func (l2Port *L2Port) setupKnetFilter(rxReason opennsl.RxReason, prio int, desc string) error {
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
	knetFilter.SetDestID(l2Port.knetNetIfaceID)
	knetFilter.SetPriority(prio)
	knetFilter.SetRxReason(rxReason)
	knetFilter.SetIngPort(l2Port.port)
	if err := knetFilter.Create(l2Port.asic.unit); err != nil {
		return err
	}

	l2Port.knetFilters[desc] = knetFilter.ID()
	return nil
}

func (l2Port *L2Port) Create(prio int) error {
	if err := l2Port.setupKnetNetIface(); err != nil {
		return err
	}

	if err := l2Port.setupKnetFilter(opennsl.RxReasonBpdu, prio, "Catch BPDU"); err != nil {
		return err
	}

	err := opennsl.PortFloodBlockSet(l2Port.asic.unit, l2Port.port, opennsl.Port(0), opennsl.PORT_FLOOD_BLOCK_UNKNOWN_UCAST)
	if err != nil {
		return err
	}

	return nil
}

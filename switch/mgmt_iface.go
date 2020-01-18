package bcm

import (
	"errors"
	"net"
	"strings"

	"github.com/beluganos/go-opennsl/opennsl"
	log "github.com/sirupsen/logrus"
)

// MgmtIntf represents settings of switch management interface.
type MgmtIntf struct {
	asic           Asic
	ifaceName      string
	vlan           opennsl.Vlan
	macAddr        net.HardwareAddr
	ipAddr         net.IP
	l3IntfID       opennsl.L3IfaceID
	l3EgressID     opennsl.L3EgressID
	knetIntfID     opennsl.KnetNetIfaceID
	knetFilterIDs    map[string]opennsl.KnetFilterID
}

func NewMgmtIntf(ifaceName string, vlan opennsl.Vlan, macAddr net.HardwareAddr, ipAddr net.IP) *MgmtIntf {
	return &MgmtIntf{
		asic:        Asic{unit: DEFAULT_ASIC_UNIT},
		ifaceName:   ifaceName,
		vlan:        vlan,
		macAddr:     macAddr,
		ipAddr:      ipAddr,
		knetFilterIDs: make(map[string]opennsl.KnetFilterID),
	}
}

// SetIntfName name of management interface.
func (mgmtIntf *MgmtIntf) SetIntfName(ifaceName string) error {
	if len(strings.TrimSpace(ifaceName)) == 0 {
		log.Errorf("Name of management interface cannot be empty")
		return errors.New("No name of management interface")
	}

	mgmtIntf.ifaceName = ifaceName
	return nil
}

// SetVlan sets VLAN ID.
func (mgmtIntf *MgmtIntf) SetVlan(vlan uint16) error {
	var vid opennsl.Vlan = opennsl.Vlan(vlan)
	if !vid.Valid() {
		log.Errorf("There is not valid VLAN ID %hu", vlan)
		return errors.New("VLAN ID is not valid")
	}

	mgmtIntf.vlan = vid
	return nil
}

// SetMACAddr sets MAC L2 address.
func (mgmtIntf *MgmtIntf) SetMACAddr(macAddr string) error {
	hwAddr, err := net.ParseMAC(macAddr)
	if err != nil {
		log.Errorf("Failed to parse MAC address %s: %s", macAddr, err)
		return err
	}

	mgmtIntf.macAddr = hwAddr
	return nil
}

// SetIPAddr sets IP L3 address.
func (mgmtIntf *MgmtIntf) SetIPAddr(ipAddr string) error {
	var ip net.IP
	if ip = net.ParseIP(ipAddr); nil == ip {
		log.Errorf("Failed to parse IP address %s", ipAddr)
		return errors.New("Invalid IP address to parse")
	}

	mgmtIntf.ipAddr = ip
	return nil
}

func (mgmtIntf *MgmtIntf) setupL3Intf() error {
	l3Intf := opennsl.NewL3Iface()
	l3Intf.SetFlags(opennsl.NewL3Flags(
		opennsl.L3_ADD_TO_ARL, // TODO: Check if it is required
		opennsl.L3_WITH_ID,
	))
	l3Intf.SetVID(mgmtIntf.vlan)
	l3Intf.SetMAC(mgmtIntf.macAddr)
	l3Intf.SetIfaceID(opennsl.L3IfaceID(3))
	if err := l3Intf.Create(mgmtIntf.asic.unit); err != nil {
		return err
	}

	mgmtIntf.l3IntfID = l3Intf.IfaceID()
	return nil
}

func (mgmtIntf *MgmtIntf) setupL3Egress() error {
	l3eg := opennsl.NewL3Egress()
	l3eg.SetIfaceID(mgmtIntf.l3IntfID)
	l3Flags := opennsl.NewL3Flags(
		opennsl.L3_COPY_TO_CPU,
		opennsl.L3_L2TOCPU,
	)
	l3eg.SetFlags(l3Flags)

	var l3egID opennsl.L3EgressID
	l3EgressID, err := l3eg.Create(mgmtIntf.asic.unit, opennsl.L3_NONE, l3egID)
	if err != nil {
		return err
	}

	mgmtIntf.l3EgressID = l3EgressID
	return nil
}

func (mgmtIntf *MgmtIntf) setupL3Host() error {
	l3Host := opennsl.NewL3Host()
	l3Host.SetIPAddr(mgmtIntf.ipAddr)
	l3Host.SetEgressID(mgmtIntf.l3EgressID)

	if err := l3Host.Add(mgmtIntf.asic.unit); err != nil {
		return err
	}

	return nil
}

func (mgmt *MgmtIntf) setupKnetNetIntf() error {
	knetNetIntf := opennsl.NewKnetNetIface()
	knetNetIntf.SetType(opennsl.KNET_NETIF_T_TX_CPU_INGRESS)
	knetNetIntf.SetVlan(opennsl.VLAN_ID_DEFAULT)
	knetNetIntf.SetName(mgmt.ifaceName)
	knetNetIntf.SetMAC(mgmt.macAddr)
	if err := knetNetIntf.Create(mgmt.asic.unit); err != nil {
		return err
	}

	mgmt.knetIntfID = knetNetIntf.ID()
	return nil
}

func (mgmt *MgmtIntf) setupKnetFilter(rxReason opennsl.RxReason, prio opennsl.KnetFilterPrio, desc string) error {
	knetFilter := opennsl.NewKnetFilter()
	knetFilter.SetDescription(desc)
	knetFilter.SetType(opennsl.KNET_FILTER_T_RX_PKT)
	knetFilter.SetFlags(opennsl.NewKnetFilterFlags(
		opennsl.KNET_FILTER_F_STRIP_TAG,
	))
	knetFilter.SetMatchFlags(opennsl.NewKnetFilterMatchFlags(
		opennsl.KNET_FILTER_M_REASON,
	))
	knetFilter.SetDestType(opennsl.KNET_DEST_T_NETIF)
	knetFilter.SetDestID(mgmt.knetIntfID)
	knetFilter.SetPriority(prio)
	knetFilter.SetRxReason(rxReason)
	if err := knetFilter.Create(mgmt.asic.unit); err != nil {
		log.Errorf("Failed to create KNET filter: %s", err)
		return err
	}

	mgmt.knetFilterIDs[desc] = knetFilter.ID()
	return nil
}

// Create creates instance of switch management interface.
func (mgmtIntf *MgmtIntf) Create() error {
	if err := mgmtIntf.setupL3Intf(); err != nil {
		return err
	}

	if err := mgmtIntf.setupL3Egress(); err != nil {
		return err
	}

	if err := mgmtIntf.setupL3Host(); err != nil {
		return err
	}

	if err := mgmtIntf.setupKnetNetIntf(); err != nil {
		return err
	}

	if err := mgmtIntf.setupKnetFilter(opennsl.RxReasonNhop, opennsl.KNET_FILTER_PRIO_NHOP, "Next Hop Packets"); err != nil {
		return err
	}

	if err := mgmtIntf.setupKnetFilter(opennsl.RxReasonProtocol, opennsl.KNET_FILTER_PRIO_PROTOCOL, "Protocol Packets"); err != nil {
		return err
	}

	return nil
}

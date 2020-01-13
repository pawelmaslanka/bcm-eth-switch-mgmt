package bcm

import (
	"errors"
	"net"
	"strings"

	"github.com/beluganos/go-opennsl/opennsl"
	log "github.com/sirupsen/logrus"
)

type mgmtIfaceKnetFiltersType map[string]int

// MgmtIface represents settings of switch management interface.
type MgmtIface struct {
	asic           Asic
	ifaceName      string
	vlan           opennsl.Vlan
	macAddr        net.HardwareAddr
	ipAddr         net.IP
	l3IfaceID      opennsl.L3IfaceID
	l3EgressID     opennsl.L3EgressID
	knetNetIfaceID int
	knetFilters    mgmtIfaceKnetFiltersType
}

func NewMgmtIface(ifaceName string, vlan opennsl.Vlan, macAddr net.HardwareAddr, ipAddr net.IP) *MgmtIface {
	return &MgmtIface{
		asic:        Asic{unit: DEFAULT_ASIC_UNIT},
		ifaceName:   ifaceName,
		vlan:        vlan,
		macAddr:     macAddr,
		ipAddr:      ipAddr,
		knetFilters: make(mgmtIfaceKnetFiltersType),
	}
}

// SetIfaceName name of management interface.
func (mgmtIface *MgmtIface) SetIfaceName(ifaceName string) error {
	if len(strings.TrimSpace(ifaceName)) == 0 {
		log.Errorf("Name of management interface cannot be empty")
		return errors.New("No name of management interface")
	}

	mgmtIface.ifaceName = ifaceName
	return nil
}

// SetVlan sets VLAN ID.
func (mgmtIface *MgmtIface) SetVlan(vlan uint16) error {
	var vid opennsl.Vlan = opennsl.Vlan(vlan)
	if !vid.Valid() {
		log.Errorf("There is not valid VLAN ID %hu", vlan)
		return errors.New("VLAN ID is not valid")
	}

	mgmtIface.vlan = vid
	return nil
}

// SetMACAddr sets MAC L2 address.
func (mgmtIface *MgmtIface) SetMACAddr(macAddr string) error {
	hwAddr, err := net.ParseMAC(macAddr)
	if err != nil {
		log.Errorf("Failed to parse MAC address %s: %s", macAddr, err)
		return err
	}

	mgmtIface.macAddr = hwAddr
	return nil
}

// SetIPAddr sets IP L3 address.
func (mgmtIface *MgmtIface) SetIPAddr(ipAddr string) error {
	var ip net.IP
	if ip = net.ParseIP(ipAddr); nil == ip {
		log.Errorf("Failed to parse IP address %s", ipAddr)
		return errors.New("Invalid IP address to parse")
	}

	mgmtIface.ipAddr = ip
	return nil
}

func (mgmtIface *MgmtIface) setupL3Iface() error {
	l3Iface := opennsl.NewL3Iface()
	l3Iface.SetFlags(opennsl.NewL3Flags(
		opennsl.L3_ADD_TO_ARL, // TODO: Check if it is required
		opennsl.L3_WITH_ID,
	))
	l3Iface.SetVID(mgmtIface.vlan)
	l3Iface.SetMAC(mgmtIface.macAddr)
	l3Iface.SetIfaceID(opennsl.L3IfaceID(3))
	if err := l3Iface.Create(mgmtIface.asic.unit); err != nil {
		return err
	}

	mgmtIface.l3IfaceID = l3Iface.IfaceID()
	return nil
}

func (mgmtIface *MgmtIface) setupL3Egress() error {
	l3eg := opennsl.NewL3Egress()
	l3eg.SetIfaceID(mgmtIface.l3IfaceID)
	l3Flags := opennsl.NewL3Flags(
		opennsl.L3_COPY_TO_CPU,
		opennsl.L3_L2TOCPU,
	)
	l3eg.SetFlags(l3Flags)

	var l3egID opennsl.L3EgressID
	l3EgressID, err := l3eg.Create(mgmtIface.asic.unit, opennsl.L3_NONE, l3egID)
	if err != nil {
		return err
	}

	mgmtIface.l3EgressID = l3EgressID
	return nil
}

func (mgmtIface *MgmtIface) setupL3Host() error {
	l3Host := opennsl.NewL3Host()
	l3Host.SetIPAddr(mgmtIface.ipAddr)
	l3Host.SetEgressID(mgmtIface.l3EgressID)

	if err := l3Host.Add(mgmtIface.asic.unit); err != nil {
		return err
	}

	return nil
}

func (mgmt *MgmtIface) setupKnetNetIface() error {
	knetNetIface := opennsl.NewKnetNetIface()
	knetNetIface.SetType(opennsl.KNET_NETIF_T_TX_CPU_INGRESS)
	knetNetIface.SetVlan(opennsl.VLAN_ID_DEFAULT)
	knetNetIface.SetName(mgmt.ifaceName)
	knetNetIface.SetMAC(mgmt.macAddr)
	if err := knetNetIface.Create(mgmt.asic.unit); err != nil {
		return err
	}

	mgmt.knetNetIfaceID = knetNetIface.ID()
	return nil
}

func (mgmt *MgmtIface) setupKnetFilter(rxReason opennsl.RxReason, prio int, desc string) error {
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
	knetFilter.SetDestID(mgmt.knetNetIfaceID)
	knetFilter.SetPriority(prio)
	knetFilter.SetRxReason(rxReason)
	if err := knetFilter.Create(mgmt.asic.unit); err != nil {
		log.Errorf("Failed to create KNET filter: %s", err)
		return err
	}

	mgmt.knetFilters[desc] = knetFilter.ID()
	return nil
}

// Create creates instance of switch management interface.
func (mgmtIface *MgmtIface) Create() error {
	if err := mgmtIface.setupL3Iface(); err != nil {
		return err
	}

	if err := mgmtIface.setupL3Egress(); err != nil {
		return err
	}

	if err := mgmtIface.setupL3Host(); err != nil {
		return err
	}

	if err := mgmtIface.setupKnetNetIface(); err != nil {
		return err
	}

	if err := mgmtIface.setupKnetFilter(opennsl.RxReasonNhop, 55, "Next Hop Packets"); err != nil {
		return err
	}

	if err := mgmtIface.setupKnetFilter(opennsl.RxReasonProtocol, 60, "Protocol Packets"); err != nil {
		return err
	}

	return nil
}

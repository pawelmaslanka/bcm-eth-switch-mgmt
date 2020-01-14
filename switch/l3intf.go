package bcm

import (
	"fmt"

	"github.com/beluganos/go-opennsl/opennsl"
)

type knetIntfFiltersType map[string]int

type L3Intf struct {
	asic        Asic
	port        opennsl.Port
	knetIntfID  int
	knetFilters knetIntfFiltersType
}

func NewL3Intf(port opennsl.Port, knetIntfID int) *L3Intf {
	return &L3Intf{
		asic:        Asic{unit: DEFAULT_ASIC_UNIT},
		knetIntfID:  knetIntfID,
		port:        port,
		knetFilters: make(knetIntfFiltersType),
	}
}

func (l3Intf *L3Intf) setupKnetFilter(rxReason opennsl.RxReason, prio int, desc string) error {
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
	knetFilter.SetDestID(l3Intf.knetIntfID)
	knetFilter.SetPriority(prio)
	knetFilter.SetRxReason(rxReason)
	knetFilter.SetIngPort(l3Intf.port)
	if err := knetFilter.Create(l3Intf.asic.unit); err != nil {
		return err
	}

	l3Intf.knetFilters[desc] = knetFilter.ID()
	return nil
}

func (l3Intf *L3Intf) Create(prio int) error {
	if err := l3Intf.setupKnetFilter(opennsl.RxReasonNhop, prio, fmt.Sprintf("Next Hop Packets %d")); err != nil {
		return err
	}

	if err := l3Intf.setupKnetFilter(opennsl.RxReasonProtocol, prio+1, fmt.Sprintf("Protocol Packets %d")); err != nil {
		return err
	}

	return nil
}

package bcm

import (
	"sync"

	"github.com/beluganos/go-opennsl/opennsl"
	log "github.com/sirupsen/logrus"
)

type void struct{}

// Asic represents per Broadcom ASIC chip parameters.
type Asic struct {
	unit int
}

// Switch represents configured parameters in Broadcom network switch layer.
type Switch struct {
	access   sync.Mutex
	asic     Asic
	stg      opennsl.Stg
	lagIntfs map[string]*LAG
}

func NewSwitch() *Switch {
	return &Switch{
		access:   sync.Mutex{},
		asic:     Asic{unit: DEFAULT_ASIC_UNIT},
		stg:      opennsl.Stg(1),
		lagIntfs: make(map[string]*LAG),
	}
}

const (
	DEFAULT_ASIC_UNIT = 0
	NUM_OF_PORTS      = 32
	PORT_1            = "eth-1"
	PORT_2            = "eth-2"
	PORT_3            = "eth-3"
	PORT_4            = "eth-4"
	PORT_5            = "eth-5"
	PORT_6            = "eth-6"
	PORT_7            = "eth-7"
	PORT_8            = "eth-8"
	PORT_9            = "eth-9"
	PORT_10           = "eth-10"
	PORT_11           = "eth-11"
	PORT_12           = "eth-12"
	PORT_13           = "eth-13"
	PORT_14           = "eth-14"
	PORT_15           = "eth-15"
	PORT_16           = "eth-16"
	PORT_17           = "eth-17"
	PORT_18           = "eth-18"
	PORT_19           = "eth-19"
	PORT_20           = "eth-20"
	PORT_21           = "eth-21"
	PORT_22           = "eth-22"
	PORT_23           = "eth-23"
	PORT_24           = "eth-24"
	PORT_25           = "eth-25"
	PORT_26           = "eth-26"
	PORT_27           = "eth-27"
	PORT_28           = "eth-28"
	PORT_29           = "eth-29"
	PORT_30           = "eth-30"
	PORT_31           = "eth-31"
	PORT_32           = "eth-32"
)

type Port_NameMap struct {
	Port     opennsl.Port
	PortName string
}

var PortNames = [NUM_OF_PORTS]Port_NameMap{
	{opennsl.Port(68), PORT_1},
	{opennsl.Port(72), PORT_2},
	{opennsl.Port(76), PORT_3},
	{opennsl.Port(80), PORT_4},
	{opennsl.Port(34), PORT_5},
	{opennsl.Port(38), PORT_6},
	{opennsl.Port(42), PORT_7},
	{opennsl.Port(46), PORT_8},
	{opennsl.Port(50), PORT_9},
	{opennsl.Port(54), PORT_10},
	{opennsl.Port(58), PORT_11},
	{opennsl.Port(62), PORT_12},
	{opennsl.Port(84), PORT_13},
	{opennsl.Port(88), PORT_14},
	{opennsl.Port(92), PORT_15},
	{opennsl.Port(96), PORT_16},
	{opennsl.Port(102), PORT_17},
	{opennsl.Port(106), PORT_18},
	{opennsl.Port(110), PORT_19},
	{opennsl.Port(114), PORT_20},
	{opennsl.Port(1), PORT_21},
	{opennsl.Port(5), PORT_22},
	{opennsl.Port(9), PORT_23},
	{opennsl.Port(13), PORT_24},
	{opennsl.Port(17), PORT_25},
	{opennsl.Port(21), PORT_26},
	{opennsl.Port(25), PORT_27},
	{opennsl.Port(29), PORT_28},
	{opennsl.Port(118), PORT_29},
	{opennsl.Port(122), PORT_30},
	{opennsl.Port(126), PORT_31},
	{opennsl.Port(130), PORT_32},
}

var NamePortIdxMap = map[string]uint16{
	PORT_1:  0,
	PORT_2:  1,
	PORT_3:  2,
	PORT_4:  3,
	PORT_5:  4,
	PORT_6:  5,
	PORT_7:  6,
	PORT_8:  7,
	PORT_9:  8,
	PORT_10: 9,
	PORT_11: 10,
	PORT_12: 11,
	PORT_13: 12,
	PORT_14: 13,
	PORT_15: 14,
	PORT_16: 15,
	PORT_17: 16,
	PORT_18: 17,
	PORT_19: 18,
	PORT_20: 19,
	PORT_21: 20,
	PORT_22: 21,
	PORT_23: 22,
	PORT_24: 23,
	PORT_25: 24,
	PORT_26: 25,
	PORT_27: 26,
	PORT_28: 27,
	PORT_29: 28,
	PORT_30: 29,
	PORT_31: 30,
	PORT_32: 31,
}

// var PortNameMap [NUM_OF_PORTS]Port_NameMap

// func init() {
// 	PortNameMap = [NUM_OF_PORTS]Port_NameMap{
// 		{opennsl.Port(68), PORT_1},
// 		{opennsl.Port(72), PORT_2},
// 		{opennsl.Port(76), PORT_3},
// 		{opennsl.Port(80), PORT_4},
// 		{opennsl.Port(34), PORT_5},
// 		{opennsl.Port(38), PORT_6},
// 		{opennsl.Port(42), PORT_7},
// 		{opennsl.Port(46), PORT_8},
// 		{opennsl.Port(50), PORT_9},
// 		{opennsl.Port(54), PORT_10},
// 		{opennsl.Port(58), PORT_11},
// 		{opennsl.Port(62), PORT_12},
// 		{opennsl.Port(84), PORT_13},
// 		{opennsl.Port(88), PORT_14},
// 		{opennsl.Port(92), PORT_15},
// 		{opennsl.Port(96), PORT_16},
// 		{opennsl.Port(102), PORT_17},
// 		{opennsl.Port(106), PORT_18},
// 		{opennsl.Port(110), PORT_19},
// 		{opennsl.Port(114), PORT_20},
// 		{opennsl.Port(1), PORT_21},
// 		{opennsl.Port(5), PORT_22},
// 		{opennsl.Port(9), PORT_23},
// 		{opennsl.Port(13), PORT_24},
// 		{opennsl.Port(17), PORT_25},
// 		{opennsl.Port(21), PORT_26},
// 		{opennsl.Port(25), PORT_27},
// 		{opennsl.Port(29), PORT_28},
// 		{opennsl.Port(118), PORT_29},
// 		{opennsl.Port(122), PORT_30},
// 		{opennsl.Port(126), PORT_31},
// 		{opennsl.Port(130), PORT_32},
// 	}
// }

func (sw *Switch) EnableFeatures() error {
	sw.access.Lock()
	defer sw.access.Unlock()

	if err := opennsl.SwitchControlsSet(
		sw.asic.unit,
		opennsl.SwitchL3EgressMode.Arg(opennsl.TRUE),
		opennsl.SwitchL3SlowpathToCpu.Arg(opennsl.TRUE),
		opennsl.SwitchArpReplyToCpu.Arg(opennsl.TRUE),
		opennsl.SwitchArpRequestToCpu.Arg(opennsl.TRUE),
	); err != nil {
		log.Errorf("Failed to set switch controlling: %s", err)
		return err
	}

	hc, err := opennsl.SwitchHashControl.Get(sw.asic.unit)
	if err != nil {
		log.Errorf("Failed to get switch hash control: %s", err)
		return err
	}

	hashControl := opennsl.HashControls(hc)
	hashControl = opennsl.NewHashControls(
		hashControl,
		opennsl.HASH_CONTROL_TRUNK_NUC_DST,
		opennsl.HASH_CONTROL_TRUNK_NUC_SRC,
		opennsl.HASH_CONTROL_TRUNK_UC_SRCPORT,
	)

	if err := opennsl.SwitchHashControl.Set(sw.asic.unit, int(hashControl)); err != nil {
		log.Errorf("Failed to set switch hash control for trunk: %s", err)
		return err
	}

	hashControl = opennsl.NewHashControls(
		hashControl,
		opennsl.HASH_CONTROL_MULTIPATH_L4PORTS,
		opennsl.HASH_CONTROL_MULTIPATH_DIP,
	)

	if err := opennsl.SwitchHashControl.Set(sw.asic.unit, int(hashControl)); err != nil {
		log.Errorf("Failed to set switch hash control for multipath: %s", err)
		return err
	}

	return nil
}

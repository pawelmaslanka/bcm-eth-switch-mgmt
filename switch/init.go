package bcm

import (
	"github.com/beluganos/go-opennsl/examples/util"
	"github.com/beluganos/go-opennsl/opennsl"
	"github.com/beluganos/go-opennsl/sal"

	log "github.com/sirupsen/logrus"
)

// Init initializes Broadcom network switch chip to default settings.
func (sw *Switch) Init() error {
	log.SetLevel(log.DebugLevel)

	if err := sal.DriverInit(); err != nil {
		log.Errorf("Failed to initialize BCM network switch driver: %s", err)
		return err
	}

	if err := opennsl.TrunkInit(sw.asic.unit); err != nil {
		log.Errorf("Failed to initialize the trunk module: %s", err)
		return err
	}

	if err := util.PortDefaultConfig(sw.asic.unit); err != nil {
		log.Errorf("Failed to apply default configuration for ports: %s", err)
		return err
	}

	if err := util.SwitchDefaultVlanConfig(sw.asic.unit); err != nil {
		log.Errorf("Failed to apply default configuration for VLAN. %s", err)
		return err
	}

	// TODO Check if we have to create default VLAN after call SwitchDefaultVlanConfig()
	defaultVlan := opennsl.Vlan(opennsl.VLAN_ID_DEFAULT)
	pcfg, err := opennsl.PortConfigGet(sw.asic.unit)
	if err != nil {
		log.Errorf("Failed to get port configuration: %s", err)
		return err
	}

	cpuBmp, _ := pcfg.PBmp(opennsl.PORT_CONFIG_CPU)
	if err := opennsl.VLAN_ID_DEFAULT.PortAdd(sw.asic.unit, cpuBmp, cpuBmp); err != nil {
		log.Errorf("Failed to add CPU port to default VLAN: %s", err)
		return err
	}

	if _, err := defaultVlan.Create(sw.asic.unit); err != nil {
		log.Errorf("Failed to create default VLAN: %s", err)
		return err
	}

	if err := defaultVlan.PortAdd(sw.asic.unit, cpuBmp, cpuBmp); err != nil {
		log.Errorf("Failed to add CPU port to default VLAN: %s", err)
		return err
	}

	return nil
}

// Release terminates use of Broadcom network switch chip.
func (sw *Switch) Release() {
	sal.DriverExit()
}

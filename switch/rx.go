package bcm

import "github.com/beluganos/go-opennsl/opennsl"

type Rx struct {
	cfg *opennsl.RxCfg
}

func NewRx() *Rx {
	return &Rx{}
}

func (rx *Rx) Start() error {
	if active := opennsl.RxActive(DEFAULT_ASIC_UNIT); !active {
		cfg := opennsl.NewRxCfg()
		cfg.SetPktSize(16 * 1024)
		cfg.SetPktsPerChain(16)
		cfg.SetGlobalPps(200)
		cfg.ChanCfg(1).SetChains(4)
		cfg.ChanCfg(1).SetCosBmp(0xffffffff)

		if err := opennsl.RxStart(DEFAULT_ASIC_UNIT, cfg); err != nil {
			return err
		}

		rx.cfg = cfg
	}

	return nil
}

func (rx *Rx) Stop() error {
	return rx.cfg.Stop(DEFAULT_ASIC_UNIT)
}

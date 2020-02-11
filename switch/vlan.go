package bcm

import (
	pb "bcm-eth-switch-mgmt/grpc_services/vlan"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/beluganos/go-opennsl/opennsl"
	log "github.com/sirupsen/logrus"
)

const (
	vlanMgmtPort = ":50056"
)

type vlanMgmtReq struct {
	pb.UnimplementedVlanMgmtServer
	sw *Switch
}

func (vlanMgmt *vlanMgmtReq) SetNativeVlan(ctx context.Context, req *pb.NativeVlan) (*pb.VlanMgmtResult, error) {
	ifname := req.GetInterface().GetIfname()
	log.Infof("Set native VLAN %d on interface %s", req.GetVid(), ifname)
	var portNames []string
	var portIdx uint16
	var exists bool
	if strings.Contains(ifname, "team") {
		log.Printf("Requested set STP state on LAG %s", ifname)
		var lag *LAG
		if lag, exists = stpMgmt.sw.lagIntfs[ifname]; !exists {
			errMsg := fmt.Sprintf("LAG %s does not exist", ifname)
			log.Errorf(errMsg)
			return &pb.StpResult{Result: pb.StpResult_FAILED}, fmt.Errorf(errMsg)
		}

		log.Printf("LAG %s has %d members", ifname, len(lag.members))
		portNames = make([]string, len(lag.members))
		i := 0
		for portName := range lag.members {
			log.Printf("Adding port %s from LAG %s", portName, ifname)
			portNames[i] = portName
			i++
		}
	} else {
		portNames = []string{ifname}
	}

	log.Printf("There are %d port names", len(portNames))
	for _, portName := range portNames {
		log.Printf("Setting STP state on port %s", portName)
		if portIdx, exists = NamePortIdxMap[portName]; !exists {
			errMsg := fmt.Sprintf("Port %s does not exist", portName)
			log.Errorf(errMsg)
			return &pb.StpResult{Result: pb.StpResult_FAILED}, fmt.Errorf(errMsg)
		}

		port := PortNames[portIdx].Port

		var stgStpState opennsl.StgStp
		switch st := state.GetState(); st {
		case pb.StpState_DISABLED:
			stgStpState = opennsl.STG_STP_DISABLE
		case pb.StpState_BLOCKING:
			stgStpState = opennsl.STG_STP_BLOCK
		case pb.StpState_LISTENING:
			stgStpState = opennsl.STG_STP_LISTEN
		case pb.StpState_LEARNING:
			stgStpState = opennsl.STG_STP_LEARN
		case pb.StpState_FORWARDING:
			stgStpState = opennsl.STG_STP_FORWARD
		default:
			log.Warnf("STG STP state %d not recognized", st)
			return &pb.StpResult{Result: pb.StpResult_FAILED}, errors.New(fmt.Sprintf("Invalid STG STP state %s for port %s", pb.StpState_State_name[int32(st)], portName))
		}

		var stg opennsl.Stg
		var err error
		if stg, err = opennsl.StpDefaultGet(DEFAULT_ASIC_UNIT); err != nil {
			log.Errorf("Failed to get default STG STP")
			return &pb.StpResult{Result: pb.StpResult_FAILED}, errors.New("Failed to get default STG STP")
		}

		if err = stg.StpSet(stpMgmt.sw.asic.unit, port, stgStpState); err != nil {
			log.Errorf("Failed to set STG STP state %d on port %s (%d)", stgStpState, portName, port)
			return &pb.StpResult{Result: pb.StpResult_FAILED}, errors.New(fmt.Sprintf("Failed to set STG STP state %s on port %s",
				pb.StpState_State_name[int32(state.GetState())], portName))
		}
	}

	return &pb.StpResult{Result: pb.StpResult_SUCCESS}, nil
}

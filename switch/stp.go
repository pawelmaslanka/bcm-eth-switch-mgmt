//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto
package bcm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	pb "OpenNosPluginForMstpd/gRPCServices"

	"github.com/beluganos/go-opennsl/opennsl"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	stpMgmtPort = ":50051"
)

// stpRequestMgmt is used to implement helloworld.GreeterServer.
type stpRequestMgmt struct {
	pb.UnimplementedStpManagementServer
	sw *Switch
}

func (stpMgmt *stpRequestMgmt) SetInterfaceState(ctx context.Context, state *pb.StpState) (*pb.StpResult, error) {
	ifname := state.GetInterface().GetIfname()
	log.Infof("SetInterfaceState: Ifname %s, state %d", ifname, state.GetState())
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

// SayHello implements helloworld.GreeterServer
func (stpMgmt *stpRequestMgmt) FlushFdb(ctx context.Context, iface *pb.StpInterface) (*pb.StpResult, error) {
	ifname := iface.GetIfname()
	log.Infof("FlushFdb on ifname %s",ifname )
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

	for _, portName := range portNames {
		log.Printf("Flushing FDB on port %s", portName)
		if portIdx, exists = NamePortIdxMap[portName]; !exists {
			errMsg := fmt.Sprintf("Port %s does not exist", portName)
			log.Errorf(errMsg)
			return &pb.StpResult{Result: pb.StpResult_FAILED}, fmt.Errorf(errMsg)
		}

		port := PortNames[portIdx].Port

		err := opennsl.L2AddrDeleteByPort(DEFAULT_ASIC_UNIT, opennsl.Module(-1), port, opennsl.NewL2DeleteFlags(opennsl.L2_DELETE_PENDING, opennsl.L2_DELETE_NO_CALLBACKS))
		if err != nil {
			log.Errorf("Failed to flush FDB on interface %s (%d)", ifname, port)
			return &pb.StpResult{Result: pb.StpResult_FAILED}, errors.New(fmt.Sprintf("Failed to flush FDB on interface %s (%d)", ifname, port))
		}
	}

	return &pb.StpResult{Result: pb.StpResult_SUCCESS}, nil
}

func (stpMgmt *stpRequestMgmt) SetAgeingTime(ctx context.Context, age *pb.StpAgeingTime) (*pb.StpResult, error) {
	log.Infof("SetAgeingTime for %u", age.AgeingTime)
	if err := opennsl.L2AddrAgeTimerSet(DEFAULT_ASIC_UNIT, int(age.AgeingTime)); err != nil {
		log.Errorf("Failed to set ageing of L2 address (%u)", age.AgeingTime)
		return &pb.StpResult{Result: pb.StpResult_FAILED}, errors.New(fmt.Sprintf("Failed to set ageing of L2 address (%u)", age.AgeingTime))
	}

	return &pb.StpResult{Result: pb.StpResult_SUCCESS}, nil
}

func HandleSTPRequest(sw *Switch) {
	lis, err := net.Listen("tcp", stpMgmtPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterStpManagementServer(s, &stpRequestMgmt{sw: sw})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// TODO: Create KNET for LAG interface? Przeciez nie potrzebujemy, bo i tak LACP ramki przychodza na podporty tego LAGowego interfejsu

package bcm

import (
	pb "OpenNosTeamdPlugin/gRPCServices"
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/beluganos/go-opennsl/opennsl"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	lagMgmtPort = ":50052"
)

type LAG struct {
	trunk   opennsl.Trunk
	members map[string]struct{}
}

func NewLAG(tid opennsl.Trunk) *LAG {
	return &LAG{
		trunk:   tid,
		members: make(map[string]struct{}),
	}
}

type lagMgmtRequest struct {
	pb.UnimplementedLagManagementServer
	sw *Switch
}

func (lagMgmt *lagMgmtRequest) CreateLag(ctx context.Context, req *pb.LagIface) (*pb.RpcResult, error) {
	lagIfname := req.GetName()
	if _, ok := lagMgmt.sw.lagIfaces[lagIfname]; ok {
		return &pb.RpcResult{Result: pb.RpcResult_SUCCESS}, nil
	}

	trunk, err := opennsl.TrunkCreate(lagMgmt.sw.asic.unit, opennsl.NewTrunkFlags(opennsl.TRUNK_FLAG_NONE))
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create LAG %s: %s", lagIfname, err)
		log.Errorf(errMsg)
		return &pb.RpcResult{Result: pb.RpcResult_FAILED}, fmt.Errorf(errMsg)
	}

	// trunkInfo := opennsl.NewTrunkInfo()
	// trunkInfo.SetDLFIndex(int(opennsl.TRUNK_UNSPEC_INDEX))
	// trunkInfo.SetMCIndex(int(opennsl.TRUNK_UNSPEC_INDEX))
	// trunkInfo.SetIPMCIndex(int(opennsl.TRUNK_UNSPEC_INDEX))
	// err = trunk.MemberSet(lagMgmt.sw.asic.unit, trunkInfo, make([]opennsl.TrunkMember, 0))
	// if err != nil {
	// 	trunk.Destroy(lagMgmt.sw.asic.unit)
	// 	errMsg := fmt.Sprintf("Failed to set trunk parameters of LAG %s: %s", lagIfname, err)
	// 	log.Errorf(errMsg)
	// 	return &pb.RpcResult{Result: pb.RpcResult_FAILED}, fmt.Errorf(errMsg)
	// }

	// TODO: Replace raw value of 9 with constant TRUNK_PSC_PORTFLOW
	err = trunk.PscSet(lagMgmt.sw.asic.unit, opennsl.TrunkPsc(9))
	if err != nil {
		trunk.Destroy(lagMgmt.sw.asic.unit)
		errMsg := fmt.Sprintf("Failed to set PSC of LAG %s: %s", lagIfname, err)
		log.Errorf(errMsg)
		return &pb.RpcResult{Result: pb.RpcResult_FAILED}, fmt.Errorf(errMsg)
	}

	lagMgmt.sw.lagIfaces[lagIfname] = NewLAG(trunk)
	return &pb.RpcResult{Result: pb.RpcResult_SUCCESS}, nil
}

// TODO: This method should set flag TRUNK_MEMBER_EGRESS_DISABLE and shoul be unset when LACP is in stae 0x3D on
//       added ports
func (lagMgmt *lagMgmtRequest) AddLagMembers(ctx context.Context, req *pb.LagMembers) (*pb.RpcResult, error) {
	var lag *LAG
	var exists bool
	lagIfname := req.GetIface().GetName()
	if lag, exists = lagMgmt.sw.lagIfaces[lagIfname]; !exists {
		errMsg := fmt.Sprintf("LAG %s does not exist", lagIfname)
		log.Errorf(errMsg)
		return &pb.RpcResult{Result: pb.RpcResult_FAILED}, fmt.Errorf(errMsg)
	}

	log.Printf("Adding ports to LAG %s", lagIfname)
	var portIdx uint16
	portMembers := req.GetMembers()
	if portMembers != nil {
		log.Printf("Number of ports: %d", len(portMembers))
	} else {
		log.Printf("Members array is empty")
		errMsg := fmt.Sprintf("Members array is empty")
		log.Errorf(errMsg)
		return &pb.RpcResult{Result: pb.RpcResult_FAILED}, fmt.Errorf(errMsg)
	}

	for _, member := range portMembers {
		portName := member.GetName()
		if len(strings.TrimSpace(portName)) == 0 {
			// No more port mebers to handle
			break
		}
		log.Printf("Adding port %s", portName)
		if _, exists = lag.members[portName]; exists {
			log.Printf("Port already exists in LAG")
			// If port is already attached to trunk, take appropriate action.
			// BCM API doesn't check for duplicate ports in trunk.
			continue
		}

		if portIdx, exists = NamePortIdxMap[member.GetName()]; !exists {
			errMsg := fmt.Sprintf("Port %s does not exist", member.GetName())
			log.Errorf(errMsg)
			return &pb.RpcResult{Result: pb.RpcResult_FAILED}, fmt.Errorf(errMsg)
		}

		port := PortNames[portIdx].Port
		gport := opennsl.GPortFromLocal(port)
		trunkMember := opennsl.NewTrunkMember()
		trunkMember.SetGPort(gport)
		if err := lag.trunk.MemberAdd(lagMgmt.sw.asic.unit, trunkMember); err != nil {
			// TODO: Let's rollback already added ports to LAG
			errMsg := fmt.Sprintf("Port %s does not exist", member.GetName())
			log.Errorf(errMsg)
			return &pb.RpcResult{Result: pb.RpcResult_FAILED}, fmt.Errorf(errMsg)
		}

		lag.members[portName] = struct{}{}
	}

	return &pb.RpcResult{Result: pb.RpcResult_SUCCESS}, nil
}

func HandleLAGRequest(sw *Switch) {
	lis, err := net.Listen("tcp", lagMgmtPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterLagManagementServer(s, &lagMgmtRequest{sw: sw})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

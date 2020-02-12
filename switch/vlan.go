package bcm

import (
	pb "bcm-eth-switch-mgmt/grpc_services/vlan"
	"context"
	"fmt"
	"net"

	"github.com/beluganos/go-opennsl/opennsl"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	vlanMgmtTcpPort = ":50056"
)

type void struct{}

type vlanMgmtReq struct {
	pb.UnimplementedVlanMgmtServer
	sw *Switch
	portsByVlan map[uint16]map[string]void
}

func (vlanMgmt *vlanMgmtReq) SetNativeVlan(ctx context.Context, req *pb.NativeVlan) (*pb.VlanMgmtResult, error) {
	// Sanity check for validation of ports which will be added to VLAN 
	var bcmPorts []opennsl.Port = make([]opennsl.Port, len(req.GetPorts()))
	var bcmPortIdx uint16
	var exists bool
	for _, port := range req.GetPorts() {
		log.Infof("Set native VLAN on port %s", port.GetName())
		if bcmPortIdx, exists = NamePortIdxMap[port.GetName()]; !exists {
			errMsg := fmt.Sprintf("Port %s does not exist", port.GetName())
			log.Errorf(errMsg)
			return &pb.VlanMgmtResult{Result: pb.VlanMgmtResult_FAILED}, fmt.Errorf(errMsg)
		}
		
		bcmPorts = append(bcmPorts, PortNames[bcmPortIdx].Port)
	}

	// Native VLAN doesn't have to exist
	// if _, exists := vlanMgmt.portsByVlan[req.GetVid()]; !exists {
		
	// }

	log.Infof("Set native VLAN %d on the following ports", req.GetVid())
	var vid opennsl.Vlan = opennsl.Vlan(req.GetVid())
	for _, bcmPort := range bcmPorts {
		if err := bcmPort.UntaggedVlanSet(vlanMgmt.sw.asic.unit, vid); err != nil {
			errMsg := fmt.Sprintf("Failed to set native VLAN %d on port %d", vid, bcmPort)
			log.Errorf(errMsg)
			return &pb.VlanMgmtResult{Result: pb.VlanMgmtResult_FAILED}, fmt.Errorf(errMsg)
		}
	}

	return &pb.VlanMgmtResult{Result: pb.VlanMgmtResult_FAILED}, nil
}

func HandleVlanMgmtRequest(sw *Switch) {
	lis, err := net.Listen("tcp", vlanMgmtTcpPort)
	if err != nil {
		log.Fatalf("Failed to listen on TCP port %d for VLAN management request: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterVlanMgmtServer(s, &vlanMgmtReq{sw: sw})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
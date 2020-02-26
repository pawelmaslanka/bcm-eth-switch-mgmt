package bcm

import (
	pb "bcm-eth-switch-mgmt/grpc_services/l3-mgmt"
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/beluganos/go-opennsl/opennsl"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	kRouteMgmtTcpPort          = ":50057"
	kNexthopsByRoute4MapSize   = 150000
	kNexthopsByNetmask4MapSize = 150000
	kRoutesByNexthop4MapSize   = 150000
	kNetmasksByRoute4MapSize   = 150000
)

type nexthopsByNetmask4Map map[uint8]uint32
type nexthopsByRoute4Map map[uint32]nexthopsByNetmask4Map
type netmasksByRoute4Map map[uint32]uint8
type routesByNexthop4Map map[uint32]netmasksByRoute4Map
type hostByNexthop4Map map[uint32]*opennsl.L3Host

// type hostByNexthop4Map map[uint32]struct { host opennsl.L3Host, count uint32 }

type routeManager struct {
	// TODO: Change it to stream server
	pb.UnimplementedRouteMgmtServer
	sw               *Switch
	cpuEgressId      opennsl.L3EgressID
	nexthopsByRoute4 nexthopsByRoute4Map
	routesByNexthop4 routesByNexthop4Map
	// TODO: Add counter for nexthop in order to delete if counter == 0
	hostByNexthop4 hostByNexthop4Map
}

func newRouteManager(sw *Switch) *routeManager {
	// Let's create default L3 egress object for software routing
	l3egToCpu := opennsl.NewL3Egress()
	l3Flags := opennsl.NewL3Flags(
		opennsl.L3_L2TOCPU,
	)
	l3egToCpu.SetFlags(l3Flags)
	var l3egID opennsl.L3EgressID

	sw.access.Lock()
	defer sw.access.Unlock()

	l3EgressID, err := l3egToCpu.Create(sw.asic.unit, opennsl.L3_NONE, l3egID)
	if err != nil {
		log.Fatalf("Failed to create CPU egress object: %s", err)
	}

	// Install default route 0.0.0.0/0 in VRF 0. It is required by ALPM mode.
	_, defaultIpNet, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := opennsl.NewL3Route()
	defaultRoute.SetIP4Net(defaultIpNet)
	defaultRoute.SetEgressID(l3EgressID)
	if err := defaultRoute.Add(sw.asic.unit); err != nil {
		log.Fatalf("Failed to install default route in TCAM: %s", err)
	}

	nextHopByRoute4 := make(nexthopsByRoute4Map)
	routeByGw4 := make(routesByNexthop4Map)
	hostByNexthop4 := make(hostByNexthop4Map)
	return &routeManager{
		sw:               sw,
		cpuEgressId:      l3EgressID,
		nexthopsByRoute4: nextHopByRoute4,
		routesByNexthop4: routeByGw4,
		hostByNexthop4:   hostByNexthop4,
	}
}

func (routeMngr *routeManager) AddRoute4(ctx context.Context, req *pb.AddRoute4Request) (*pb.RouteMgmtResult, error) {
	networkCidrReq := req.GetNetwork().GetCidr4()
	// log.Infof("Requested adding route %s", networkCidrReq)
	networkIpReq, networkIpNetReq, err := net.ParseCIDR(networkCidrReq)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to parse network %s", networkCidrReq)
		log.Errorf(errMsg)
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_FAILED}, fmt.Errorf(errMsg)
	}

	nextHopIpReq := req.GetNextHop().GetIp4()
	// log.Infof("Requested adding route via next hop %s", nextHopIpReq)
	nextHopIp := net.ParseIP(nextHopIpReq)
	if nextHopIp == nil {
		errMsg := fmt.Sprintf("Failed to parse gateway IP %s", nextHopIpReq)
		log.Errorf(errMsg)
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_FAILED}, fmt.Errorf(errMsg)
	}

	ipSubnetMaskSize, bitsInMask := networkIpNetReq.Mask.Size()
	if ipSubnetMaskSize == 0 && bitsInMask == 0 {
		errMsg := fmt.Sprintf("Failed to get network prefix length for network %s", networkIpNetReq.String())
		log.Errorf(errMsg)
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_FAILED}, fmt.Errorf(errMsg)
	}

	networkPrefix := uint8(ipSubnetMaskSize)
	networkIp := ip4ToInt(networkIpReq)
	if _, exists := routeMngr.nexthopsByRoute4[networkIp]; !exists {
		// log.Infof("Route does not exists in cache")
		routeMngr.nexthopsByRoute4[networkIp] = make(nexthopsByNetmask4Map)
	} else if _, exists := routeMngr.nexthopsByRoute4[networkIp][networkPrefix]; exists {
		// log.Infof("Next hop already exists")
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_SUCCESS}, nil
	}

	nextHopIpAsInt := ip4ToInt(nextHopIp)
	if _, exists := routeMngr.hostByNexthop4[nextHopIpAsInt]; !exists {
		// TODO: Now assume that gateway is not resolved, send packet to CPU for software routing
		l3Host := opennsl.NewL3Host()
		l3Host.SetIPAddr(nextHopIp)
		l3Host.SetEgressID(routeMngr.cpuEgressId)
		if err := l3Host.Add(routeMngr.sw.asic.unit); err != nil {
			errMsg := fmt.Sprintf("Failed to install L3 host info %s in ASIC: %s", nextHopIp.String(), err)
			log.Errorf(errMsg)
			return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_FAILED}, fmt.Errorf(errMsg)
		}

		routeMngr.hostByNexthop4[nextHopIpAsInt] = l3Host
	}

	route := opennsl.NewL3Route()
	route.SetIP4Net(networkIpNetReq)
	route.SetEgressID(routeMngr.cpuEgressId)

	routeMngr.sw.access.Lock()
	defer routeMngr.sw.access.Unlock()

	if err := route.Add(routeMngr.sw.asic.unit); err != nil {
		errMsg := fmt.Sprintf("Failed to install L3 route %s in ASIC: %s", networkIpNetReq.String(), err)
		log.Errorf(errMsg)
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_FAILED}, fmt.Errorf(errMsg)
	}

	// If added new route to TCAM with success, save new gateway in cache
	routeMngr.nexthopsByRoute4[networkIp][networkPrefix] = nextHopIpAsInt

	return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_SUCCESS}, nil
}

func (routeMngr *routeManager) WithdrawRoute4(ctx context.Context, req *pb.WithdrawRoute4Request) (*pb.RouteMgmtResult, error) {
	networkCidrReq := req.GetNetwork().GetCidr4()
	// log.Infof("Requested withdraw network %s", networkCidrReq)
	networkIpReq, networkIpNetReq, err := net.ParseCIDR(networkCidrReq)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to parse network %s", networkCidrReq)
		log.Errorf(errMsg)
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_FAILED}, fmt.Errorf(errMsg)
	}

	ipSubnetMaskSize, bitsInMask := networkIpNetReq.Mask.Size()
	if ipSubnetMaskSize == 0 && bitsInMask == 0 {
		errMsg := fmt.Sprintf("Failed to get network prefix length for network %s", networkIpNetReq.String())
		log.Errorf(errMsg)
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_FAILED}, fmt.Errorf(errMsg)
	}

	networkPrefix := uint8(ipSubnetMaskSize)
	networkIp := ip4ToInt(networkIpReq)
	if _, exists := routeMngr.nexthopsByRoute4[networkIp]; !exists {
		// log.Infof("Route does not exists in cache")
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_SUCCESS}, nil
	}

	if _, exists := routeMngr.nexthopsByRoute4[networkIp][networkPrefix]; !exists {
		// log.Infof("Next hop not exists in cache")
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_SUCCESS}, nil
	}

	route := opennsl.NewL3Route()
	route.SetIP4Net(networkIpNetReq)
	route.SetEgressID(routeMngr.cpuEgressId)

	routeMngr.sw.access.Lock()
	defer routeMngr.sw.access.Unlock()

	if err := route.Delete(routeMngr.sw.asic.unit); err != nil {
		errMsg := fmt.Sprintf("Failed to delete L3 route %s from ASIC: %s", networkIpNetReq.String(), err)
		log.Errorf(errMsg)
		return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_FAILED}, fmt.Errorf(errMsg)
	}

	delete(routeMngr.nexthopsByRoute4, networkIp)
	return &pb.RouteMgmtResult{Result: pb.RouteMgmtResult_SUCCESS}, nil
}

func HandleRequestOfL3RouteMgmtRpc(sw *Switch) {
	lis, err := net.Listen("tcp", kRouteMgmtTcpPort)
	if err != nil {
		log.Fatalf("Failed to listen on TCP port %d for L3 route management request: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterRouteMgmtServer(s, newRouteManager(sw))
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve L3 route management request: %v", err)
	}
}

func ip4ToInt(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func intToIp4(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

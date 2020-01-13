################################################################################
#
# bcm-eth-switch-mgmt
#
################################################################################
BCM_ETH_SWITCH_MGMT_VERSION = 1.0
BCM_ETH_SWITCH_MGMT_SITE = $(BR2_EXTERNAL_DESTINY_PATH)/package/bcm-eth-switch-mgmt
BCM_ETH_SWITCH_MGMT_SITE_METHOD = local
BCM_ETH_SWITCH_MGMT_LICENSE = Apache-2.0
# BCM_ETH_SWITCH_MGMT_LICENSE_FILES = LICENSE
BCM_ETH_SWITCH_MGMT_DEPENDENCIES = host-go go-opennsl

define BCM_ETH_SWITCH_MGMT_POST_RSYNC_HOOK
	# GOPATH=$(@D)/_gopath ${GO_BIN} get -u github.com/sirupsen/logrus
	mkdir -p $(@D)/_gopath/{bin,pkg,src}
	mkdir -p $(@D)/_gopath/src/OpenNosPluginForMstpd/gRPCServices
	mkdir -p $(@D)/_gopath/src/OpenNosTeamdPlugin/gRPCServices
	GOPATH=$(@D)/_gopath ${GO_BIN} get -u google.golang.org/grpc
	cp -r $(@D)/gRPCServices/stp_management* $(@D)/_gopath/src/OpenNosPluginForMstpd/gRPCServices
	cp -r $(@D)/gRPCServices/lag_management* $(@D)/_gopath/src/OpenNosTeamdPlugin/gRPCServices
	cp -rf ${GO_OPENNSL_DIR}/_gopath/src/* $(@D)/_gopath/src
	cp -rf ${GO_OPENNSL_DIR}/_gopath/pkg/* $(@D)/_gopath/pkg
	mkdir -p $(@D)/_gopath/src/bcm-eth-switch-mgmt
	mv $(@D)/switch $(@D)/_gopath/src/bcm-eth-switch-mgmt
endef

BCM_ETH_SWITCH_MGMT_POST_RSYNC_HOOKS += BCM_ETH_SWITCH_MGMT_POST_RSYNC_HOOK

define BCM_ETH_SWITCH_MGMT_BUILD_CMDS
	cd $(@D); \
	CGO_CFLAGS=-I/workdir/buildconfig/br_output/host/x86_64-buildroot-linux-gnu/sysroot/usr/include/bcm-opennsl/ \
	GO111MODULE=off \
	GOARCH=amd64 \
	GOCACHE="/workdir/buildconfig/br_output/host/usr/share/go-cache" \
	GOROOT="/workdir/buildconfig/br_output/host/lib/go" \
	CC="/workdir/buildconfig/br_output/host/bin/x86_64-unknown-linux-gnu-gcc" \
	CXX="/workdir/buildconfig/br_output/host/bin/x86_64-unknown-linux-gnu-g++" \
	GOTOOLDIR="/workdir/buildconfig/br_output/host/lib/go/pkg/tool/linux_amd64" \
	PATH="/workdir/buildconfig/br_output/host/bin:/workdir/buildconfig/br_output/host/sbin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
	GOBIN= CGO_ENABLED=1 \
	CGO_CFLAGS="${CGO_CFLAGS} -I/workdir/buildconfig/br_output/host/x86_64-buildroot-linux-gnu/sysroot/usr/include/opennsl" \
	CGO_LDFLAGS="-L/workdir/buildconfig/br_output/host/x86_64-buildroot-linux-gnu/sysroot/usr/lib -lopennsl" \
	GOPATH=$(@D)/_gopath ${GO_BIN} build -o bcm-eth-switch-mgmt
endef

define BCM_ETH_SWITCH_MGMT_INSTALL_TARGET_CMDS
	cp $(@D)/bcm-eth-switch-mgmt $(TARGET_DIR)/usr/bin
endef

$(eval $(generic-package))
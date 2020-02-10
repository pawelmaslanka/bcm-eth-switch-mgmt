#!/bin/bash
protoc -I . ./vlan.proto --go_out=plugins=grpc:.
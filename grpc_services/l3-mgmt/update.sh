#!/bin/bash
protoc -I . ./route.proto --go_out=plugins=grpc:.
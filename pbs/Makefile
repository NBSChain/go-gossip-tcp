SHELL=PATH='$(PATH)' /bin/sh

.PHONY: all

all:
	protoc --go_out=plugins=grpc:. *.proto

SHELL=PATH='$(PATH)' /bin/sh

PLATFORM := $(shell uname -o)

EXTEND := .exe
ifeq ($(PLATFORM), Msys)
    INCLUDE := ${shell echo "$(GOPATH)"|sed -e 's/\\/\//g'}
else ifeq ($(PLATFORM), Cygwin)
    INCLUDE := ${shell echo "$(GOPATH)"|sed -e 's/\\/\//g'}
else
	INCLUDE := $(GOPATH)
	EXTEND	:=
endif

EXENAME := gsp$(EXTEND)

# enable second expansion
.SECONDEXPANSION:

	echo $(PLATFORM)

all: pbs build

build:
	go build -race -o $(EXENAME)
	mv $(EXENAME) $(INCLUDE)/bin/

pbs:
	cd ../pbs/ && $(MAKE)

CGO_ENABLED: 0
GOFLAGS: -mod=vendor
PWD ?= $(shell pwd)

build:
	GOBIN=${PWD} go install -a -v \
	    -tags='osusergo netgo static static_build' \
	    -ldflags="-s -w '-extldflags=-static'" \
	    -installsuffix='netgo' \
	    github.com/kouzoh/merlin

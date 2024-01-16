# Run this manually upon any .proto file(s) change(s).
# The generated .pb.go is checked into repo.
# Else calling this Makefile from lom's Makefile breaks compilation.
# Even in case of sonic-gnmi, generationhappens manually.
#
ifeq ($(GOPATH),)
export GOPATH=/tmp/go
endif

GOBIN_PATH := /usr/local/go1.20.3/go/bin
GO := $(GOBIN_PATH)/go

export PATH := $(PATH):$(GOPATH)/bin:$(GOBIN_PATH)

PROTOC_OPTS := -I$(CURDIR)/vendor -I/usr/local/include -I/usr/include

PROTOC_GEN_GO := $(GOPATH)/bin/protoc-gen-go

go.mod:
	$(GO) mod init test
	$(GO) mod tidy          # updates go.mod with a line like "require github.com/openconfig/gnmi"
	$(GO) mod vendor        # Get packages into vendor folder
	$(info go.mod done)
	@echo go.mod done via echo


$(PROTOC_GEN_GO):
	cd $$(mktemp -d) && \
	$(GO) mod init protoc && \
	$(GO) get github.com/golang/protobuf/protoc-gen-go && \
	$(GO) install github.com/golang/protobuf/protoc-gen-go 
	FILES="$(shell ls)"; echo $$FILES
	@echo Done protoc-gen-go

	    
# TODO: Improve it to auto handle multiple proto files.
# PROTO_GO_BINDINGS = proto/sonic_internal.pb.go
# $(PROTO_GO_BINDINGS): $$(patsubst %.pb.go,%.proto,$$@) $(PROTOC_GEN_GO)
	# protoc -I$(@D) $(PROTOC_OPTS) --go_out=plugins=grpc:$(@D) $<

PROTOC_OPTS := -I$(CURDIR)/vendor -I/usr/local/include -I/usr/include

lom_internal_pb.go: lom_internal.proto go.mod $(PROTOC_GEN_GO)
	protoc -I$(@D) $(PROTOC_OPTS) --go_out=plugins=grpc:$(@D) lom_internal.proto

doClean:
	rm -f go.mod go.sum
	rm -rf vendor

all: lom_internal_pb.go doClean





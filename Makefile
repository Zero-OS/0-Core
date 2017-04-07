OUTPUT = bin
VERSION = base/version.go

branch = $(shell git rev-parse --abbrev-ref HEAD)
revision = $(shell git rev-parse HEAD)
dirty = $(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
base = github.com/g8os/core0/base
ldflags0 = '-w -s -X $(base).Branch=$(branch) -X $(base).Revision=$(revision) -X $(base).Dirty=$(dirty)'
ldflagsX = '-w -s -X $(base).Branch=$(branch) -X $(base).Revision=$(revision) -X $(base).Dirty=$(dirty) -extldflags "-static"'

all: core0 coreX

core0: $(OUTPUT)
	cd core0 && go build -ldflags $(ldflags0) -o ../$(OUTPUT)/$@

coreX: $(OUTPUT)
	cd coreX && CGO_ENABLED=0 GOOS=linux go build -ldflags $(ldflagsX) -o ../$(OUTPUT)/$@


$(OUTPUT):
	mkdir -p $(OUTPUT)

.PHONY: $(OUTPUT) core0 coreX

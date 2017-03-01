OUTPUT = bin
VERSION = base/version.go

all: version core0 coreX clean

.PHONEY: all

core0: $(OUTPUT) $(VERSION)
	cd core0 && go build -o ../$(OUTPUT)/$@

coreX: $(OUTPUT) $(VERSION)
	cd coreX && go build -o ../$(OUTPUT)/$@

$(OUTPUT):
	mkdir -p $(OUTPUT)

version: branch = $(shell git rev-parse --abbrev-ref HEAD)
version: revision = $(shell git rev-parse HEAD)
version: dirty = $(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
version:
	sed -i 's/{branch}/$(branch)/' $(VERSION)
	sed -i 's/{revision}/$(revision)/' $(VERSION)
	sed -i 's/{dirty}/$(dirty)/' $(VERSION)

clean:
	git checkout $(VERSION)
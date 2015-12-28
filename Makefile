REPONAME = collectors
VERSION = dev

.PHONY: all build_release 

all: build_release

build_release:
	$(MAKE) -C ./files/zip

test:
	$(MAKE) -C ./files/zip
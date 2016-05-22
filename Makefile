REPONAME = collectors
VERSION = dev

.PHONY: all build_release 

all: build_release

build_release:
	$(MAKE) -C ./files/zip
	$(MAKE) -C ./files/faban
	$(MAKE) -C ./dbms/mysql
	$(MAKE) -C ./environment/logs
	$(MAKE) -C ./environment/stats
	$(MAKE) -C ./environment/properties

test:
	$(MAKE) -C ./files/zip
	$(MAKE) -C ./files/faban
	$(MAKE) -C ./dbms/mysql
	$(MAKE) -C ./environment/logs
	$(MAKE) -C ./environment/stats
	$(MAKE) -C ./environment/properties

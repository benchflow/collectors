REPONAME = collectors
VERSION = dev

.PHONY: all build_release 

all: build_release

build_release:
	$(MAKE) -C ./files/zip
	$(MAKE) -C ./dbms/mysql
	$(MAKE) -C ./environment/logs
	$(MAKE) -C ./environment/stats

test:
	$(MAKE) -C ./files/zip
	$(MAKE) -C ./dbms/mysql
	$(MAKE) -C ./environment/logs
	$(MAKE) -C ./environment/stats
REPONAME = collectors
VERSION = dev

.PHONY: all build_release 

all: build_release

build_release:
	$(MAKE) -C ./files/zip
	$(MAKE) -C ./dbms/mysql/mysqldump

test:
	$(MAKE) -C ./files/zip
	$(MAKE) -C ./dbms/mysql/mysqldump
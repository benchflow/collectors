REPONAME = collectors
VERSION = dev

.PHONY: all build_release

all: build_release

build_release:
	$(MAKE) -C ./files/zip
	$(MAKE) -C ./dbms/mysql
	$(MAKE) -C ./environment/logs
	$(MAKE) -C ./environment/stats
	$(MAKE) -C ./environment/properties

build_container:
	$(MAKE) -C ./files/zip build_container
	$(MAKE) -C ./dbms/mysql build_container
	$(MAKE) -C ./environment/logs build_container
	$(MAKE) -C ./environment/stats build_container
	$(MAKE) -C ./environment/properties build_container

get_commons_dependencies:
	$(MAKE) -C ./files/zip get_commons_dependencies
	$(MAKE) -C ./dbms/mysql get_commons_dependencies
	$(MAKE) -C ./environment/logs get_commons_dependencies
	$(MAKE) -C ./environment/stats get_commons_dependencies
	$(MAKE) -C ./environment/properties get_commons_dependencies

test: build_release
	$(MAKE) -C ./files/zip test
	$(MAKE) -C ./dbms/mysql test
	$(MAKE) -C ./environment/logs test
	$(MAKE) -C ./environment/stats test
	$(MAKE) -C ./environment/properties test

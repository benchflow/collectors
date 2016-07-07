REPONAME = collectors
NAME = mysql
DOCKERIMAGENAME = benchflow/$(REPONAME):$(NAME)
VERSION = dev
GOPATH_SAVE_RESTORE:=$(shell pwd):${GOPATH}

.PHONY: all get_commons_dependencies build_release 

all: get_commons_dependencies build_release

save_dependencies:
	# TODO: figure out how to get the following work, currently we add the dependencies to the Godeps.json manually
	# - refer to the following issue: https://github.com/benchflow/benchflow/issues/33
	# cd src/cloud/benchflow/$(REPONAME)/ && \
	# GOPATH=$(GOPATH_SAVE_RESTORE) godep save ./... && \
	# rm -rf ../../../../Godeps/*.* && \
	# rm -rf ../../../../Godeps && \
	# mv Godeps/ ../../../.. && \
	# cd ../../../..

get_commons_dependencies:
	# TODO: decide how to get also the commons' dependencies from the relatinve Godeps file in the commons repo
	# - refer to the following issue: https://github.com/benchflow/benchflow/issues/33
	mkdir -p ./Godeps/_workspace/src/github.com/benchflow/commons/kafka/
	wget -O ./Godeps/_workspace/src/github.com/benchflow/commons/kafka/kafka.go https://github.com/benchflow/commons/releases/download/v-dev/kafka.go
	mkdir -p ./Godeps/_workspace/src/github.com/benchflow/commons/minio/
	wget -O ./Godeps/_workspace/src/github.com/benchflow/commons/minio/minio.go https://github.com/benchflow/commons/releases/download/v-dev/minio.go

restore_external_dependencies:
	rm -rf Godeps/_workspace/src/github.com/*
	GOPATH=$(GOPATH_SAVE_RESTORE) godep restore ./...
	# TODO: make it working with all but cloud folder
	mv src/github.com/* Godeps/_workspace/src/github.com
	rm -rf src/github.com

restore_dependencies: restore_external_dependencies get_commons_dependencies

# TODO: figure out how to get vendor dependencies committed with the standard git flow
	# - refer to the following issue: https://github.com/benchflow/benchflow/issues/33
prepare_to_commit:
	cd Godeps && \
	find . | grep .git/ | xargs rm -rf && \
	git add --all -f .

clean:
	go clean -i ./...
	rm -rf Godeps/_workspace/pkg

build:
	godep go build -o bin/$(NAME) -v ./...

build_release:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go build -ldflags '-s' -o bin/$(NAME) -v ./...

install:
	godep go install -v ./...
	mv bin/$(REPONAME) bin/$(NAME)

test:
	godep go test ./...

build_container:
	docker build -t $(DOCKERIMAGENAME)_$(VERSION) -f Dockerfile .

build_container_local:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 godep go build -ldflags '-s' -o bin/$(NAME)_linux -v ./...
	docker build -t $(DOCKERIMAGENAME)_$(VERSION) -f Dockerfile.test .
	rm bin/$(NAME)_linux

test_container_local:
	docker run -d --name db -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=stuff mysql
	docker run -ti --rm -e "ENVCONSUL_CONSUL=$(ENVCONSUL_CONSUL)" \
	-e "TRIAL_ID=1_1" -e "KAFKA_HOST=$(KAFKA_HOST)" -e "COLLECTOR_NAME=mysql" \
	-e "MYSQL_DB_NAME=stuff" -e "TABLE_NAMES=test" -e "MYSQL_HOST=10.0.2.15" \
	-e "MYSQL_PORT=3306" -e "MYSQL_USER=root" -e "MYSQL_USER_PASSWORD=password" \
	-e "CONTAINER_NAME=$(NAME)" -e "COLLECTOR_NAME=$(NAME)" -e "DATA_NAME=$(NAME)" \
	-e "MINIO_ALIAS=benchflow" -e "MINIO_HOST=$(MINIO_HOST)" -e "MINIO_ACCESSKEYID=$(MINIO_ACCESSKEYID)" \
	-e "MINIO_SECRETACCESSKEY=$(MINIO_SECRETACCESSKEY)" \
	-p 8080:8080 --name $(NAME) $(DOCKERIMAGENAME)_$(VERSION)
	#TODO
	
build_and_test_container_local: build_container_local test_container_local
	
rm_container_local:
	docker rm -f -v db
	docker rm -f -v $(NAME)

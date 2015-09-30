FROM golang
 
RUN apt-get update \
	&& wget https://dl.minio.io:9000/updates/2015/Sept/linux-amd64/mc \
	&& chmod +x mc \
    && apt-get -y install p7zip \
    && apt-get clean
 
ADD src/cloud/benchflow/collectors /go/src/cloud/benchflow/collectors

RUN go get github.com/fsouza/go-dockerclient
RUN go install cloud/benchflow/collectors

ENV MINIO_HOST https://play.minio.io:9000
ENV CONTAINERS db

ENTRYPOINT /go/bin/collectors
 
EXPOSE 8080

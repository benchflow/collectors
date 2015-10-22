FROM golang
 
RUN apt-get update \
    && apt-get -y install p7zip \
    && apt-get clean
 
ADD src/cloud/benchflow/collectors /go/src/cloud/benchflow/collectors

RUN go get github.com/fsouza/go-dockerclient
RUN go get github.com/minio/minio-go
RUN go install cloud/benchflow/collectors

ENV CONTAINER_NAME wfms-CPU
ENV CONTAINERS db
ENV MINIO_HOST http://192.168.99.100:9000
ENV MINIO_ACCESS_KEY_ID dummy
ENV MINIO_SECRET_ACCESS_KEY dummy

ENTRYPOINT /go/bin/collectors
 
EXPOSE 8080

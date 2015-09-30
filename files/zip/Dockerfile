FROM golang
 
RUN apt-get update \
	&& wget https://dl.minio.io:9000/updates/2015/Sept/linux-amd64/mc \
	&& chmod +x mc \
    && apt-get -y install p7zip \
    && apt-get clean

ADD src/cloud/benchflow/collectors /go/src/cloud/benchflow/collectors

RUN go install cloud/benchflow/collectors

ENV TO_ZIP /media:/go/src
ENV MINIO_HOST https://play.minio.io:9000

ENTRYPOINT /go/bin/collectors
 
EXPOSE 8080

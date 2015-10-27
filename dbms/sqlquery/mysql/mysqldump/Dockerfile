FROM golang
 
RUN apt-get update \
	&& apt-get -y install mysql-client-5.5 \
    && apt-get -y install gzip \
    && apt-get clean

ENV CONTAINER_NAME wfms-dbms
ENV MYSQL_HOST db
ENV MYSQL_USER root
ENV MYSQL_USER_PASSWORD PASSWORD
ENV MYSQL_DB_NAME stuff
ENV MINIO_HOST http://192.168.99.100:9000
ENV MINIO_ACCESS_KEY_ID dummy
ENV MINIO_SECRET_ACCESS_KEY dummy

ADD src/cloud/benchflow/collectors /go/src/cloud/benchflow/collectors

RUN go get github.com/minio/minio-go
RUN go install cloud/benchflow/collectors

ENTRYPOINT /go/bin/collectors
 
EXPOSE 8080

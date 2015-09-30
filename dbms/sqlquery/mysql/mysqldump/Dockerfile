FROM golang
 
RUN apt-get update \
	&& wget https://dl.minio.io:9000/updates/2015/Sept/linux-amd64/mc \
	&& chmod +x mc \
	&& apt-get -y install mysql-client-5.5 \
    && apt-get -y install p7zip \
    && apt-get clean

ENV MYSQL_HOST db
ENV MYSQL_USER root
ENV MYSQL_USER_PASSWORD PASSWORD
ENV MYSQL_DB_NAME stuff
ENV MINIO_HOST https://play.minio.io:9000

ADD src/cloud/benchflow/collectors /go/src/cloud/benchflow/collectors

RUN go install cloud/benchflow/collectors

ENTRYPOINT /go/bin/collectors
 
EXPOSE 8080

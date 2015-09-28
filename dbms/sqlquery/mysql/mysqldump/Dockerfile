FROM golang
 
RUN apt-get update \
	&& apt-get -y install mysql-client-5.5 \
    && apt-get -y install p7zip \
    && apt-get clean

ADD src/cloud/benchflow/collectors /go/src/cloud/benchflow/collectors

RUN go install cloud/benchflow/collectors

ENTRYPOINT /go/bin/collectors
 
EXPOSE 8080

FROM golang
 
RUN apt-get update \
    && apt-get -y install p7zip \
    && apt-get clean

ADD src/cloud/benchflow/collectors /go/src/cloud/benchflow/collectors

RUN go install cloud/benchflow/collectors

ENV TO_ZIP /media:/go/src

ENTRYPOINT /go/bin/collectors
 
EXPOSE 8080

FROM golang
 
ADD src/cloud/benchflow/collectors /go/src/cloud/benchflow/collectors

RUN go get github.com/fsouza/go-dockerclient
RUN go install cloud/benchflow/collectors

ENTRYPOINT /go/bin/collectors
 
EXPOSE 8080

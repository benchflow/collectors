package main

import (
	"bufio"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	//"github.com/minio/minio-go"
	"github.com/benchflow/commons/minio"
	"log"
	"net/http"
	"os"
	"strings"
	"strconv"
)

type Container struct {
	ID string
}

var containers [10]Container
var client docker.Client

func collectStats(container Container, since int64) {
	fo, _ := os.Create(container.ID + "_tmp")
	writerOut := bufio.NewWriter(fo)
	fe, _ := os.Create(container.ID + "_tmp_err")
	writerErr := bufio.NewWriter(fe)
	err := client.Logs(docker.LogsOptions{
		Container:    container.ID,
		OutputStream: writerOut,
		ErrorStream:  writerErr,
		Follow:       false,
		Stdout:       true,
		Stderr:       true,
		Since:        since,
		Timestamps:   true,
	})
	if err != nil {
		log.Fatal(err)
	}
	
	fo.Close()
	fe.Close()
	
	minio.GzipFile(container.ID+"_tmp")
	minio.GzipFile(container.ID+"_tmp_err")

	minio.StoreOnMinio(container.ID+"_tmp.gz", "runs", minio.GenerateKey("logs.gz"))
	minio.StoreOnMinio(container.ID+"_tmp_err.gz", "runs", minio.GenerateKey("logs_err.gz"))
}

func storeData(w http.ResponseWriter, r *http.Request) {
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ":")
	since := r.FormValue("since")
	sinceInt, err := strconv.ParseInt(since, 10, 64)
	if err != nil {
		panic(err)
		}
	for i, _ := range conts {
		collectStats(containers[i], sinceInt)
	}
}

func createDockerClient() docker.Client {
	path := os.Getenv("DOCKER_CERT_PATH")
	endpoint := "tcp://"+os.Getenv("DOCKER_HOST")+":2376"
	//endpoint = "tcp://192.168.99.100:2376"
    ca := fmt.Sprintf("%s/ca.pem", path)
    cert := fmt.Sprintf("%s/cert.pem", path)
    key := fmt.Sprintf("%s/key.pem", path)
    client, err := docker.NewTLSClient(endpoint, cert, key, ca)
	//endpoint := "unix:///var/run/docker.sock"
    //client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
		}
	return *client
	}

func main() {
	client = createDockerClient()
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ":")
	for i, each := range conts {
		c := Container{ID: each}
		containers[i] = c
	}
	http.HandleFunc("/store", storeData)
	http.ListenAndServe(":8080", nil)
}

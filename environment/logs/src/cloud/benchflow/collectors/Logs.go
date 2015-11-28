package main

import (
	"bufio"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/minio/minio-go"
	"log"
	"net/http"
	"os"
	"os/exec"
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
	
	gzipFile(container.ID+"_tmp")
	gzipFile(container.ID+"_tmp_err")

	storeOnMinio(container.ID+"_tmp.gz")
	storeOnMinio(container.ID+"_tmp_err.gz")
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

func gzipFile(fileName string) {
	cmd := exec.Command("gzip", fileName)
	err := cmd.Start()
	cmd.Wait()
	if err != nil {
		panic(err)
		}
	}

func storeOnMinio(fileName string) {
	config := minio.Config{
		AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("MINIO_SECRET_ACCESS_KEY"),
		Endpoint:        os.Getenv("MINIO_HOST"),
		}
		s3Client, err := minio.New(config)
	    if err != nil {
	        log.Fatalln(err)
	    }  
	    object, err := os.Open(fileName)
		if err != nil {
			log.Fatalln(err)
		}
		defer object.Close()
		objectInfo, err := object.Stat()
		if err != nil {
			object.Close()
			log.Fatalln(err)
		}
		err = s3Client.PutObject("benchmarks", fileName, "application/octet-stream", objectInfo.Size(), object)
		if err != nil {
			log.Fatalln(err)
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

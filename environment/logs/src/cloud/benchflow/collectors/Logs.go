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
)

type Container struct {
	ID string
}

var containers [10]Container
var client docker.Client

func collectStats(container Container) {
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
		Since:        0,
		Timestamps:   true,
	})
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("7zr", "a", container.ID+"_tmp.7z", container.ID+"_tmp", container.ID+"_tmp_err")
	err = cmd.Start()
	cmd.Wait()
	if err != nil {
		panic(err)
	}

	config := minio.Config{
		AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("MINIO_SECRET_ACCESS_KEY"),
		Endpoint:        os.Getenv("MINIO_HOST"),
	}
	s3Client, err := minio.New(config)
	if err != nil {
		log.Fatalln(err)
	}
	object, err := os.Open(container.ID + "_tmp.7z")
	if err != nil {
		log.Fatalln(err)
	}
	defer object.Close()
	objectInfo, err := object.Stat()
	if err != nil {
		object.Close()
		log.Fatalln(err)
	}
	err = s3Client.PutObject("data", container.ID+"_tmp.7z", "application/octet-stream", objectInfo.Size(), object)
	if err != nil {
		log.Fatalln(err)
	}
	for object := range s3Client.ListObjects("data", "", true) {
		if object.Err != nil {
			log.Fatalln(object.Err)
		}
		log.Println(object.Stat)
	}
}

func storeData(w http.ResponseWriter, r *http.Request) {
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ":")
	for i, _ := range conts {
		collectStats(containers[i])
	}
}

func main() {
	path := os.Getenv("DOCKER_CERT_PATH")
	endpoint := os.Getenv("DOCKER_HOST")
	ca := fmt.Sprintf("%s/ca.pem", path)
	cert := fmt.Sprintf("%s/cert.pem", path)
	key := fmt.Sprintf("%s/key.pem", path)
	c, err := docker.NewTLSClient(endpoint, cert, key, ca)
	/* endpoint := "unix:///var/run/docker.sock"
	   client, err := docker.NewClient(endpoint) */
	if err != nil {
		log.Fatal(err)
	}
	client = *c
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ":")
	for i, each := range conts {
		c := Container{ID: each}
		containers[i] = c
	}
	http.HandleFunc("/store", storeData)
	http.ListenAndServe(":8080", nil)
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"io"
	"bufio"
)
import "github.com/fsouza/go-dockerclient"

type Container struct {
	ID           string
	out io.Writer
	err io.Writer
}

var containers [10]Container
var collecting bool

func collectStats(client docker.Client, container Container) {
	fo, _ := os.Create(container.ID+"_tmp")
	fo.Close()
	writer := bufio.NewWriter(fo)
	writer.Flush()
	go func() {
		err := client.Logs(docker.LogsOptions{
			Container: container.ID,
			OutputStream: writer,
    		ErrorStream:  writer,
		    Follow: true,
		    Stdout: true,
		    Stderr: true,
		    Since: 0,
		    Timestamps: true,
		    Tail: "10",
		})
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func monitorStats(container *Container) {
	go func() {
		for true {
			if !collecting {
				container.out = nil
				cmd := exec.Command("./mc", "cp", container.ID+"_tmp", os.Getenv("MINIO_HOST"))
    			err := cmd.Start()
    			cmd.Wait()
    			if err != nil {
        			panic(err)
    				}
				return
			}
		}
	}()
}

func startCollecting(w http.ResponseWriter, r *http.Request) {
	if collecting {
		fmt.Fprintf(w, "Already collecting")
		return
	}
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ":")
	for i, _ := range conts {
		monitorStats(&containers[i])
	}
	collecting = true
	fmt.Fprintf(w, "Started collecting")
}

func stopCollecting(w http.ResponseWriter, r *http.Request) {
	if !collecting {
		fmt.Fprintf(w, "Currently not collecting")
		return
	}
	collecting = false
	fmt.Fprintf(w, "Stopped collecting")
}

func main() {
	collecting = false
	path := os.Getenv("DOCKER_CERT_PATH")
	endpoint := os.Getenv("DOCKER_HOST")
	ca := fmt.Sprintf("%s/ca.pem", path)
	cert := fmt.Sprintf("%s/cert.pem", path)
	key := fmt.Sprintf("%s/key.pem", path)
	client, err := docker.NewTLSClient(endpoint, cert, key, ca)
	if err != nil {
		log.Fatal(err)
	}
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ":")
	for i, each := range conts {
		c := Container{ID: each, out: nil, err: nil}
		containers[i] = c
		collectStats(*client, containers[i])
	}
	http.HandleFunc("/start", startCollecting)
	http.HandleFunc("/stop", stopCollecting)
	http.ListenAndServe(":8080", nil)
}

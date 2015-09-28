package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)
import "github.com/fsouza/go-dockerclient"

type Container struct {
	ID           string
	statsChannel chan *docker.Stats
	flagsChannel chan bool
	last5        uint64
	last5Time    time.Time
	last30       uint64
	last30Time   time.Time
	last60       uint64
	last60Time   time.Time
}

var containers [10]Container
var collecting bool

func collectStats(client docker.Client, container Container) {
	go func() {
		err := client.Stats(docker.StatsOptions{
			ID:      container.ID,
			Stats:   container.statsChannel,
			Stream:  true,
			Done:    container.flagsChannel,
			Timeout: time.Duration(10),
		})
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func monitorStats(container *Container) {
	go func() {
		var e docker.Env
		fo, _ := os.Create("tmp")
		for true {
			dat := (<-container.statsChannel)
			e.SetJSON("dat", dat)
			fo.Write([]byte(e.Get("dat")))
			fo.Write([]byte("\n \n"))
			time.Sleep(time.Second)
			if !collecting {
				fo.Close()
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
		statsChannel := make(chan *docker.Stats)
		flagsChannel := make(chan bool)
		c := Container{ID: each, statsChannel: statsChannel, flagsChannel: flagsChannel}
		containers[i] = c
		collectStats(*client, containers[i])
	}
	http.HandleFunc("/start", startCollecting)
	http.HandleFunc("/stop", stopCollecting)
	http.ListenAndServe(":8080", nil)
}

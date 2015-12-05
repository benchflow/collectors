package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"github.com/fsouza/go-dockerclient"
	//"github.com/minio/minio-go"
	"github.com/Cerfoglg/commons/src/minio"
)

type Container struct {
	ID           string
	statsChannel chan *docker.Stats
}

var containers []Container
var stopChannel chan bool
var doneChannel chan bool
var waitGroup sync.WaitGroup
var collecting bool

func attachToContainer(client docker.Client, container Container) {
	go func() {
		err := client.Stats(docker.StatsOptions{
			ID:      container.ID,
			Stats:   container.statsChannel,
			Stream:  true,
			Done:    doneChannel,
			Timeout: 0,
		})
		if err != nil {
			//log.Fatal(err)
		}
		//fmt.Println("Stopped collecting")
	}()
}

func collectStats(container Container) {
	go func() {
		var e docker.Env
		fo, _ := os.Create(container.ID+"_tmp")
		for true {
			select {
			case <- stopChannel:
				close(doneChannel)
				fo.Close()
				minio.GzipFile(container.ID+"_tmp")
				minio.StoreOnMinio(container.ID+"_tmp.gz", "runs", minio.GenerateKey("stats.gz"))
				waitGroup.Done()
				return
			default:
				dat := (<-container.statsChannel)
				e.SetJSON("dat", dat)
				fo.Write([]byte(e.Get("dat")))
				fo.Write([]byte("\n"))
				}
		}
	}()
}

func createDockerClient() docker.Client {
	//path := os.Getenv("DOCKER_CERT_PATH")
	//endpoint := "tcp://"+os.Getenv("DOCKER_HOST")+":2376"
	//endpoint := "tcp://192.168.99.100:2376"
    //ca := fmt.Sprintf("%s/ca.pem", path)
    //cert := fmt.Sprintf("%s/cert.pem", path)
    //key := fmt.Sprintf("%s/key.pem", path)
    //client, err := docker.NewTLSClient(endpoint, cert, key, ca)
	endpoint := "unix:///var/run/docker.sock"
    client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
		}
	return *client
	}

func startCollecting(w http.ResponseWriter, r *http.Request) {
	if collecting {
		fmt.Fprintf(w, "Already collecting")
		return
	}
	client := createDockerClient()
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ":")
	containers = []Container{}
	stopChannel = make(chan bool)
	doneChannel = make(chan bool)
	for _, each := range conts {
		statsChannel := make(chan *docker.Stats)
		c := Container{ID: each, statsChannel: statsChannel}
		containers = append(containers, c)
		attachToContainer(client, c)
		collectStats(c)
		waitGroup.Add(1)
	}
	collecting = true
	fmt.Fprintf(w, "Started collecting")
}

func stopCollecting(w http.ResponseWriter, r *http.Request) {
	if !collecting {
		fmt.Fprintf(w, "Currently not collecting")
		return
	}
	close(stopChannel)
	waitGroup.Wait()
	collecting = false
	fmt.Fprintf(w, "Stopped collecting")
}

func main() {
	collecting = false
	
	http.HandleFunc("/start", startCollecting)
	http.HandleFunc("/stop", stopCollecting)
	http.ListenAndServe(":8080", nil)
}
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"sync"
	"github.com/fsouza/go-dockerclient"
	"github.com/minio/minio-go"
)

type Container struct {
	ID           string
	statsChannel chan *docker.Stats
	doneChannel chan bool
}

var containers []Container
var stopChannel chan bool
var waitGroup sync.WaitGroup
var collecting bool

func attachToContainer(client docker.Client, container Container) {
	go func() {
		err := client.Stats(docker.StatsOptions{
			ID:      container.ID,
			Stats:   container.statsChannel,
			Stream:  true,
			Done:    container.doneChannel,
			Timeout: time.Duration(10),
		})
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func collectStats(container Container) {
	go func() {
		var e docker.Env
		fo, _ := os.Create(container.ID+"_tmp")
		for true {
			select {
			case <- stopChannel:
				//container.doneChannel <- true
				fo.Close()
				gzipFile(container.ID+"_tmp")
				//storeOnMinio(container.ID+"_tmp")
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
	for _, each := range conts {
		statsChannel := make(chan *docker.Stats)
		doneChannel := make(chan bool)
		c := Container{ID: each, statsChannel: statsChannel, doneChannel: doneChannel}
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

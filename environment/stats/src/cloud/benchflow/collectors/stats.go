package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"bytes"
	"strings"
	"sync"
	"github.com/fsouza/go-dockerclient"
	"github.com/Shopify/sarama"
	"encoding/json"
	//"github.com/minio/minio-go"
	"github.com/benchflow/commons/minio"
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

type KafkaMessage struct {
	SUT_name string `json:"SUT_name"`
	SUT_version string `json:"SUT_version"`
	Minio_key string `json:"minio_key"`
	Trial_id string `json:"trial_id"`
	Experiment_id string `json:"experiment_id"`
	Total_trials_num int `json:"total_trials_num"`
	}

func signalOnKafka(minioKey string) {
	kafkaMsg := KafkaMessage{SUT_name: "Camunda", SUT_version: "", Minio_key: minioKey, Trial_id: os.Getenv("TRIAL_ID"), Experiment_id: os.Getenv("EXPERIMENT_ID"), Total_trials_num: os.Getenv("TOTAL_TRIALS_NUM")}
	jsMessage, err := json.Marshal(kafkaMsg)
	if err != nil {
		log.Printf("Failed to marshall json message")
		}
	producer, err := sarama.NewSyncProducer([]string{os.Getenv("KAFKA_HOST")+":9092"}, nil)
	if err != nil {
	    log.Fatalln(err)
	}
	defer func() {
	    if err := producer.Close(); err != nil {
	        log.Fatalln(err)
	    }
	}()
	msg := &sarama.ProducerMessage{Topic: os.Getenv("COLLECTOR_NAME"), Value: sarama.StringEncoder(jsMessage)}
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
	    log.Printf("FAILED to send message: %s\n", err)
	    } else {
	    log.Printf("> message sent to partition %d at offset %d\n", partition, offset)
	    }
	}

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
		fo, err := os.Create("/app/"+container.ID+"_tmp")
		if err != nil {
	        panic(err)
	    }
		for true {
			select {
			case <- stopChannel:
				close(doneChannel)
				fo.Close()
				minio.GzipFile("/app/"+container.ID+"_tmp")
				minioKey := minio.GenerateKey(container.ID+"_stats.gz")
				callMinioClient("/app/"+container.ID+"_tmp.gz", os.Getenv("MINIO_ALIAS"), minioKey)
				//minio.StoreOnMinio(container.ID+"_tmp.gz", "runs", minioKey)
				signalOnKafka(minioKey)
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

func callMinioClient(fileName string, minioHost string, minioKey string) {
		//TODO: change, we are using sudo to elevate the priviledge in the container, but it is not nice
		//NOTE: it seems that the commands that are not in PATH, should be launched using sh -c
		log.Printf("sh -c sudo /app/mc --quiet cp " + fileName + " " + minioHost + "/runs/" + minioKey)
		cmd := exec.Command("sh", "-c", "sudo /app/mc --quiet cp " + fileName + " " + minioHost + "/runs/" + minioKey)
    	var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
		    fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		    return
		}
		fmt.Println("Result: " + out.String())
	}

func main() {
	collecting = false
	
	http.HandleFunc("/start", startCollecting)
	http.HandleFunc("/stop", stopCollecting)
	http.ListenAndServe(":8080", nil)
}
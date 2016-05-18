package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"bytes"
	"strings"
	"github.com/fsouza/go-dockerclient"
	"github.com/Shopify/sarama"
	"encoding/json"
	"github.com/benchflow/commons/minio"
	"strconv"
)

type KafkaMessage struct {
	SUT_name string `json:"SUT_name"`
	SUT_version string `json:"SUT_version"`
	Minio_key string `json:"minio_key"`
	Trial_id string `json:"trial_id"`
	Experiment_id string `json:"experiment_id"`
	Container_id string `json:"container_id"`
	Total_trials_num int `json:"total_trials_num"`
	Collector_name string `json:"collector_name"`
	}

func signalOnKafka(minioKey string, containerID string) {
	totalTrials, _ := strconv.Atoi(os.Getenv("BENCHFLOW_TRIAL_TOTAL_NUM"))
	kafkaMsg := KafkaMessage{SUT_name: os.Getenv("SUT_NAME"), SUT_version: os.Getenv("SUT_VERSION"), Minio_key: minioKey, Trial_id: os.Getenv("BENCHFLOW_TRIAL_ID"), Experiment_id: os.Getenv("BENCHFLOW_EXPERIMENT_ID"), Container_id: containerID, Total_trials_num: totalTrials, Collector_name: os.Getenv("BENCHFLOW_COLLECTOR_NAME")}
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
	msg := &sarama.ProducerMessage{Topic: os.Getenv("KAFKA_TOPIC"), Value: sarama.StringEncoder(jsMessage)}
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
	    log.Printf("FAILED to send message: %s\n", err)
	    } else {
	    log.Printf("> message sent to partition %d at offset %d\n", partition, offset)
	    }
	}

func createDockerClient() docker.Client {
	//path := os.Getenv("DOCKER_CERT_PATH")
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

func storeData(w http.ResponseWriter, r *http.Request) {
	client := createDockerClient()
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ",")
	composedMinioKey := ""
	composedContainerIds := ""
	for _, each := range conts {
		var e docker.Env
		
		foInspect, err := os.Create("/app/"+each+"_inspect_tmp")
		if err != nil {
	        panic(err)
	    }
		foInfo, err := os.Create("/app/"+each+"_info_tmp")
		if err != nil {
	        panic(err)
	    }
		foVersion, err := os.Create("/app/"+each+"_version_tmp")
		if err != nil {
	        panic(err)
	    }
		
		inspect, err := client.InspectContainer(each)
		if err != nil {
			panic(err)
			}
		e.SetJSON("inspect", inspect)
		foInspect.Write([]byte(e.Get("inspect")))
		
		info, err := client.Info()
		if err != nil {
			panic(err)
			}
		e.SetJSON("info", info)
		foInfo.Write([]byte(e.Get("info")))
		
		version, err := client.Version()
		if err != nil {
			panic(err)
			}
		e.SetJSON("version", version)
		foVersion.Write([]byte(e.Get("version")))
		
		minio.GzipFile("/app/"+each+"_inspect_tmp")
		minio.GzipFile("/app/"+each+"_info_tmp")
		minio.GzipFile("/app/"+each+"_version_tmp")
		
		minioKey := minio.GenerateKey(each)
		composedMinioKey = composedMinioKey+minioKey+","
		composedContainerIds = composedContainerIds+inspect.ID+","
		
		fmt.Println(minioKey)
		
		callMinioClient("/app/"+each+"_inspect_tmp.gz", os.Getenv("MINIO_ALIAS"), minioKey+"_inspect.gz")
		callMinioClient("/app/"+each+"_info_tmp.gz", os.Getenv("MINIO_ALIAS"), minioKey+"_info.gz")
		callMinioClient("/app/"+each+"_version_tmp.gz", os.Getenv("MINIO_ALIAS"), minioKey+"_version.gz")
		
		err = os.Remove("/app/"+each+"_inspect_tmp.gz")
		if err != nil {
	        panic(err)
	    }
		err = os.Remove("/app/"+each+"_info_tmp.gz")
		if err != nil {
	        panic(err)
	    }
		err = os.Remove("/app/"+each+"_version_tmp.gz")
		if err != nil {
	        panic(err)
	    }
	}
	composedMinioKey = strings.TrimRight(composedMinioKey, ",")
	composedContainerIds = strings.TrimRight(composedContainerIds, ",")
	fmt.Println(composedMinioKey)
	signalOnKafka(composedMinioKey, composedContainerIds)
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
	http.HandleFunc("/store", storeData)
	http.ListenAndServe(":8080", nil)
}
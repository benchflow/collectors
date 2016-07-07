package main

import (
	"bufio"
	//"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/benchflow/commons/minio"
	"github.com/benchflow/commons/kafka"
	"log"
	"net/http"
	"os"
	"strings"
	"strconv"
)

var client docker.Client

func storeData(w http.ResponseWriter, r *http.Request) {
	composedMinioKey := ""
	composedContainerIds := ""
	
	info, err := client.Info()
	if err != nil {
		panic(err)
		}
	hostID := info.ID
	
	since := r.FormValue("since")
	var sinceInt int64
	sinceInt = 0
	if since != "" {
		sinceInt, err = strconv.ParseInt(since, 10, 64)
		if err != nil {
			panic(err)
		}
	}
	
	client = createDockerClient()
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ",")
	
	for _, container := range conts {
		inspect, err := client.InspectContainer(container)
		if err != nil {
			panic(err)
			}
		
		fo, _ := os.Create(inspect.Name + "_tmp")
		writerOut := bufio.NewWriter(fo)
		fe, _ := os.Create(inspect.Name + "_tmp_err")
		writerErr := bufio.NewWriter(fe)
		err = client.Logs(docker.LogsOptions{
			Container:    inspect.ID,
			OutputStream: writerOut,
			ErrorStream:  writerErr,
			Follow:       false,
			Stdout:       true,
			Stderr:       true,
			Since:        sinceInt,
			Timestamps:   true,
		})
		if err != nil {
			log.Fatal(err)
		}
		
		fo.Close()
		fe.Close()
		
		minio.GzipFile(inspect.Name+"_tmp")
		minio.GzipFile(inspect.Name+"_tmp_err")
		
		//minioKey := minio.GenerateKey("logs.gz")
		minioKey := minio.GenerateKey(inspect.Name, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
		
		composedMinioKey = composedMinioKey+minioKey+","
		composedContainerIds = composedContainerIds+inspect.ID+","
		
		minio.SendGzipToMinio(inspect.Name+"_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+".gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		//callMinioClient(container.ID+"_tmp.gz", os.Getenv("MINIO_ALIAS"), minioKey)
		//minio.StoreOnMinio(container.ID+"_tmp.gz", "runs", minioKey)
		
		//callMinioClient(container.ID+"_tmp_err.gz", os.Getenv("MINIO_ALIAS"), minioKey)
		minio.SendGzipToMinio(inspect.Name+"_tmp_err.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_err.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		//minio.StoreOnMinio(container.ID+"_tmp_err.gz", "runs", minioKey)
		//kafka.SignalOnKafka(minioKey)
	}
	kafka.SignalOnKafka(composedMinioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), composedContainerIds, hostID, os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
}

func createDockerClient() docker.Client {
	endpoint := "unix:///var/run/docker.sock"
    client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
		}
	return *client
}

func main() {
	client = createDockerClient()
	http.HandleFunc("/store", storeData)
	http.ListenAndServe(":8080", nil)
}

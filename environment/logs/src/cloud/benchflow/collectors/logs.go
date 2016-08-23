package main

import (
	"bufio"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/benchflow/commons/minio"
	"github.com/benchflow/commons/kafka"
	"log"
	"net/http"
	"encoding/json"
	"os"
	"strings"
	"strconv"
)

type Response struct {
  Status string
  Message string
}

func writeJSONResponse(w http.ResponseWriter, status string, message string) {
	response := Response{status, message}
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println(err)
	    return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(js)	
}

var client docker.Client

func storeData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		w.WriteHeader(405)
		return	
	}
	composedMinioKey := ""
	composedContainerIds := ""
	composedContainerNames := ""
	
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err)
	    	return
		}
	}
	
	client = createDockerClient()
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ",")
	
	for _, container := range conts {
		inspect, err := client.InspectContainer(container)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err)
	    	return
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err)
			return
		}
		
		fo.Close()
		fe.Close()
		
		minio.GzipFile(inspect.Name+"_tmp")
		minio.GzipFile(inspect.Name+"_tmp_err")
		
		minioKey := minio.GenerateKey(inspect.Name, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
		
		composedMinioKey = composedMinioKey+minioKey+","
		composedContainerIds = composedContainerIds+inspect.ID+","
		cName := strings.Split(container, "_")[0]
		composedContainerNames = composedContainerNames+cName+","
		
		minio.SendGzipToMinio(inspect.Name+"_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+".gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		minio.SendGzipToMinio(inspect.Name+"_tmp_err.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_err.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
	}
	composedMinioKey = strings.TrimRight(composedMinioKey, ",")
	composedContainerIds = strings.TrimRight(composedContainerIds, ",")
	composedContainerNames = strings.TrimRight(composedContainerNames, ",")
	kafka.SignalOnKafka(composedMinioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), composedContainerIds, composedContainerNames, hostID, os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	
	writeJSONResponse(w, "SUCCESS", "The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
	fmt.Println("The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
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

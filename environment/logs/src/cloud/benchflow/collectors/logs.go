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

//Structure of the response message for the API requests
type Response struct {
  Status string
  Message string
}

//Function to marshall and write the response to the API requests
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

//The variable containg the object representing the Docker client
var client docker.Client

//Primary function that is called when a request is received from a client on /store
func storeData(w http.ResponseWriter, r *http.Request) {
	//If request method is not PUT then respond with method not allowed
	if r.Method != "PUT" {
		w.WriteHeader(405)
		return	
	}
	//Vars for the composed minio keys, container ids and names
	composedMinioKey := ""
	composedContainerIds := ""
	composedContainerNames := ""
	
	//Retrieve host id
	info, err := client.Info()
	if err != nil {
		panic(err)
		}
	hostID := info.ID
	
	//Retrieve and parse query "since" for the starting point of the logs, if available
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
	
	//Create Docker client
	client = createDockerClient()
	
	//Get container names/ids to collect from
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ",")
	
	//Iterate over containers and collect data from each
	for _, container := range conts {
		//Inspect the container via the Docker client
		inspect, err := client.InspectContainer(container)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err)
	    	return
		}
		
		//Create temp files for logs, both stdout and stderr and write logs into them
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
		
		//Gzip the files
		minio.GzipFile(inspect.Name+"_tmp")
		minio.GzipFile(inspect.Name+"_tmp_err")
		
		//Generate the minio key
		minioKey := minio.GenerateKey(inspect.Name, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
		
		//Append minio keys and containers to the combined vars
		composedMinioKey = composedMinioKey+minioKey+","
		composedContainerIds = composedContainerIds+inspect.ID+","
		cName := strings.Split(container, "_")[0]
		composedContainerNames = composedContainerNames+cName+","
		
		//Send files to Minio
		minio.SendGzipToMinio(inspect.Name+"_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+".gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		minio.SendGzipToMinio(inspect.Name+"_tmp_err.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_err.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
	}
	//Trim composed strings of the trailing coma, signal Kafka
	composedMinioKey = strings.TrimRight(composedMinioKey, ",")
	composedContainerIds = strings.TrimRight(composedContainerIds, ",")
	composedContainerNames = strings.TrimRight(composedContainerNames, ",")
	kafka.SignalOnKafka(composedMinioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), composedContainerIds, composedContainerNames, hostID, os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	
	//Write response
	writeJSONResponse(w, "SUCCESS", "The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
	fmt.Println("The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
}

//Function to create the Docker client using the Docker socket (shared when the container was run)
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

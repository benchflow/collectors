package main

import (
	"log"
	"fmt"
	"net/http"
	"os"
	"strings"
	"github.com/fsouza/go-dockerclient"
	"github.com/benchflow/commons/minio"
	"github.com/benchflow/commons/kafka"
	"encoding/json"
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

//Function to create the Docker client object to communicate with Docker, using the Docker socket (shared with the container)
func createDockerClient() docker.Client {
	endpoint := "unix:///var/run/docker.sock"
    client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	return *client
}

//Primary function that is called when a request is received from a client on /store
func storeData(w http.ResponseWriter, r *http.Request) {
	//If request method is not PUT then respond with method not allowed
	if r.Method != "PUT" {
		w.WriteHeader(405)
		return	
	}
	
	//Create Docker client
	client := createDockerClient()
	
	//Get Docker info (host properties)
	info, err := client.Info()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println(err)
    	return
	}
	
	//Get Docker version data
	version, err := client.Version()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println(err)
    	return
	}
	
	//Get list of containers to collect from
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ",")
	
	//Initiating variables for lists of keys, container ids and names
	composedMinioKey := ""
	composedContainerIds := ""
	composedContainerNames := ""
	
	//Host id
	hostID := info.ID
	
	//For all containers observed retrieve properties
	for _, each := range conts {
		//Var to store the JSONs obtained from the API
		var e docker.Env
		
		//Creating temp files
		foInspect, err := os.Create("/app/"+each+"_inspect_tmp")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err)
	    	return
		}
		foInfo, err := os.Create("/app/"+each+"_info_tmp")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err)
	    	return
		}
		foVersion, err := os.Create("/app/"+each+"_version_tmp")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err)
	    	return
		}
		
		//Inspecting container (container properties) and writing the data in the files
		inspect, err := client.InspectContainer(each)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println(err)
	    	return
		}
		e.SetJSON("inspect", inspect)
		foInspect.Write([]byte(e.Get("inspect")))
		
		e.SetJSON("info", info)
		foInfo.Write([]byte(e.Get("info")))
		
		e.SetJSON("version", version)
		foVersion.Write([]byte(e.Get("version")))
		
		//Gzipping files
		minio.GzipFile("/app/"+each+"_inspect_tmp")
		minio.GzipFile("/app/"+each+"_info_tmp")
		minio.GzipFile("/app/"+each+"_version_tmp")
		
		//Generating Minio key, adding it to the list of minio keys, same for container ids and names
		minioKey := minio.GenerateKey(each, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
		composedMinioKey = composedMinioKey+minioKey+","
		composedContainerIds = composedContainerIds+inspect.ID+","
		cName := strings.Split(each, "_")[0]
		composedContainerNames = composedContainerIds+cName+","
		
		//Sending files to Minio
		minio.SendGzipToMinio("/app/"+each+"_inspect_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_inspect.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		
		minio.SendGzipToMinio("/app/"+each+"_info_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_info.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		
		minio.SendGzipToMinio("/app/"+each+"_version_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_version.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		
		//Removing temp files
		err = os.Remove("/app/"+each+"_inspect_tmp.gz")
		if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	        fmt.Println(err)
	    	return
	    }
		err = os.Remove("/app/"+each+"_info_tmp.gz")
		if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	        fmt.Println(err)
	    	return
	    }
		err = os.Remove("/app/"+each+"_version_tmp.gz")
		if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	        fmt.Println(err)
	    	return
	    }
	}
	//Trimming trailing comas from vars
	composedMinioKey = strings.TrimRight(composedMinioKey, ",")
	composedContainerIds = strings.TrimRight(composedContainerIds, ",")
	composedContainerNames = strings.TrimRight(composedContainerNames, ",")
	
	//Signalling on Kafka
	kafka.SignalOnKafka(composedMinioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), composedContainerIds, composedContainerNames, hostID, os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	
	//Writing response
	writeJSONResponse(w, "SUCCESS", "The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
	fmt.Println("The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
}

func main() {
	http.HandleFunc("/store", storeData)
	http.ListenAndServe(":8080", nil)
}
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"github.com/fsouza/go-dockerclient"
	"github.com/benchflow/commons/minio"
	"github.com/benchflow/commons/kafka"
	"encoding/json"
)

type Response struct {
  Successful bool
}

func createDockerClient() docker.Client {
	endpoint := "unix:///var/run/docker.sock"
    client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
		}
	return *client
	}

func storeData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		w.WriteHeader(405)
		return	
	}
	client := createDockerClient()
	
	info, err := client.Info()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
	}
	
	version, err := client.Version()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
	}
	
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ",")
	composedMinioKey := ""
	composedContainerIds := ""
	composedContainerNames := ""
	hostID := info.ID
	for _, each := range conts {
		var e docker.Env
		
		foInspect, err := os.Create("/app/"+each+"_inspect_tmp")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
		}
		foInfo, err := os.Create("/app/"+each+"_info_tmp")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
		}
		foVersion, err := os.Create("/app/"+each+"_version_tmp")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
		}
		
		inspect, err := client.InspectContainer(each)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
		}
		e.SetJSON("inspect", inspect)
		foInspect.Write([]byte(e.Get("inspect")))
		
		e.SetJSON("info", info)
		foInfo.Write([]byte(e.Get("info")))
		
		e.SetJSON("version", version)
		foVersion.Write([]byte(e.Get("version")))
		
		minio.GzipFile("/app/"+each+"_inspect_tmp")
		minio.GzipFile("/app/"+each+"_info_tmp")
		minio.GzipFile("/app/"+each+"_version_tmp")
		
		minioKey := minio.GenerateKey(each, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
		composedMinioKey = composedMinioKey+minioKey+","
		composedContainerIds = composedContainerIds+inspect.ID+","
		cName := strings.Split(each, "_")[0]
		composedContainerNames = composedContainerIds+cName+","
		
		minio.SendGzipToMinio("/app/"+each+"_inspect_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_inspect.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		
		minio.SendGzipToMinio("/app/"+each+"_info_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_info.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		
		minio.SendGzipToMinio("/app/"+each+"_version_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_version.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		
		err = os.Remove("/app/"+each+"_inspect_tmp.gz")
		if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }
		err = os.Remove("/app/"+each+"_info_tmp.gz")
		if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }
		err = os.Remove("/app/"+each+"_version_tmp.gz")
		if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }
	}
	composedMinioKey = strings.TrimRight(composedMinioKey, ",")
	composedContainerIds = strings.TrimRight(composedContainerIds, ",")
	composedContainerNames = strings.TrimRight(composedContainerNames, ",")
	
	kafka.SignalOnKafka(composedMinioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), composedContainerIds, composedContainerNames, hostID, os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	
	response := Response{true}
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	    return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
}

func main() {
	http.HandleFunc("/store", storeData)
	http.ListenAndServe(":8080", nil)
}
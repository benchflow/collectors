package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "github.com/benchflow/commons/minio"
    "github.com/benchflow/commons/kafka"
    "encoding/json"
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
 
func backupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		w.WriteHeader(405)
		return	
	}
    ev := os.Getenv("TO_ZIP")
    paths := strings.Split(ev, ",")
    minioKey := minio.GenerateKey("folders", os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
    for _, each := range paths {
        fmt.Fprintf(w, "Trying to zip %s\n", each)
        folderList := strings.Split(each, "/")
	    folderName := folderList[len(folderList)-1]
        cmd := exec.Command("tar", "-zcvf", folderName+".tar.gz", each)
		err := cmd.Start()
		cmd.Wait()
		if err != nil {
			panic(err)
		}  
		minio.SendGzipToMinio(folderName+".tar.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+folderName+".tar.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
	}
    kafka.SignalOnKafka(minioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), "files", "files", "host", os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	
	writeJSONResponse(w, "SUCCESS", "The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
	fmt.Println("The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
}

func main() {
    http.HandleFunc("/store", backupHandler)
    http.ListenAndServe(":8080", nil)
}
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

//Primary function that is called when a request is received from a client on /store
func backupHandler(w http.ResponseWriter, r *http.Request) {
	//If request method is not PUT then respond with method not allowed
	if r.Method != "PUT" {
		w.WriteHeader(405)
		return	
	}
	//Get the list of folders to zip (coma separated list)
    ev := os.Getenv("TO_ZIP")
    paths := strings.Split(ev, ",")
    minioKey := minio.GenerateKey("folders", os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
    //For each folder, tar it, and send it to Minio
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
    //Signal on Kafka
    kafka.SignalOnKafka(minioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), "files", "files", "host", os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	
	//Write response to client
	writeJSONResponse(w, "SUCCESS", "The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
	fmt.Println("The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
}

func main() {
    http.HandleFunc("/store", backupHandler)
    http.ListenAndServe(":8080", nil)
}
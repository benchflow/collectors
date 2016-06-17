package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "github.com/benchflow/commons/minio"
    "github.com/benchflow/commons/kafka"
)
 
func backupHandler(w http.ResponseWriter, r *http.Request) {
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
	    //minioKey := minio.GenerateKey(folderName+".tar.gz")
		//callMinioClient(folderName+".tar.gz", os.Getenv("MINIO_ALIAS"), minioKey)
		minio.SendGzipToMinio(folderName+".tar.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+folderName+".tar.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
	    //minio.StoreOnMinio(folderName+".tar.gz", "runs", minioKey)
	}
    kafka.SignalOnKafka(minioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), "files", "host", os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	fmt.Fprintf(w, "SUCCESS")
}

func main() {
    http.HandleFunc("/store", backupHandler)
    http.ListenAndServe(":8080", nil)
}
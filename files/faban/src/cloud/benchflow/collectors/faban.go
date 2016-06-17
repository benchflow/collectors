package main
 
import (
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "os/exec"
    "log"
    "github.com/benchflow/commons/minio"
    "github.com/benchflow/commons/kafka"
)

var fabanOutputRoot = "/app/faban/output/"
 
func backupHandler(w http.ResponseWriter, r *http.Request) {
    fabanRunId := r.FormValue("faban_run_id")
    path := fabanOutputRoot+fabanRunId
    files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	paths := []string{}
    for _, file := range files {
		paths = append(paths, path+"/"+file.Name())
	}
	cmd := exec.Command("gzip", paths...)
	err = cmd.Start()
	cmd.Wait()
	if err != nil {
		panic(err)
		} 
	minioKey := minio.GenerateKey(fabanRunId, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
    for _, each := range files { 
		//callMinioClient(path+"/"+each.Name()+".gz", os.Getenv("MINIO_ALIAS"), minioKey+"/"+each.Name()+".gz")
		minio.SendGzipToMinio(path+"/"+each.Name()+".gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"/"+each.Name()+".gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
	}
    //kafka.SignalOnKafka(minioKey, "faban")
    kafka.SignalOnKafka(minioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), "faban", "host", os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	fmt.Fprintf(w, "SUCCESS")
}
 
func main() {
    http.HandleFunc("/store", backupHandler)
    http.ListenAndServe(":8080", nil)
}
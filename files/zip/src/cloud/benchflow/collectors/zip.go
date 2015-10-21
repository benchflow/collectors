package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "github.com/minio/minio-go"
    "log"
)
 
func backupHandler(w http.ResponseWriter, r *http.Request) {
    ev := os.Getenv("TO_ZIP")
    paths := strings.Split(ev, ":")
    for _, each := range paths {
        fmt.Fprintf(w, "Trying to zip %s\n", each)
    }
    zipCommand := strings.Split("a,/folders.7z", ",")
    paths = append(zipCommand, paths...)
    cmd := exec.Command("7zr", paths...)
    err := cmd.Start()
    cmd.Wait()
    if err != nil {
        fmt.Fprintf(w, "ERROR:  %s", err)
        panic(err)
    }
    
    config := minio.Config{
        AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY_ID"),
        SecretAccessKey: os.Getenv("MINIO_SECRET_ACCESS_KEY"),
        Endpoint:        os.Getenv("MINIO_HOST"),
    }
    s3Client, err := minio.New(config)
    if err != nil {
        log.Fatalln(err)
    }
    object, err := os.Open("/folders.7z")
	if err != nil {
		log.Fatalln(err)
	}
	defer object.Close()
	objectInfo, err := object.Stat()
	if err != nil {
		object.Close()
		log.Fatalln(err)
	}
	err = s3Client.PutObject("benchmarks/a/runs/1", os.Getenv("CONTAINER_NAME")+"_zipping.7z", "application/octet-stream", objectInfo.Size(), object)
	if err != nil {
		log.Fatalln(err)
	}
	for object := range s3Client.ListObjects("folders", "", true) {
		if object.Err != nil {
			log.Fatalln(object.Err)
		}
		log.Println(object.Stat)
	}
	
	fmt.Fprintf(w, "SUCCESS")
}
 
func main() {
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}
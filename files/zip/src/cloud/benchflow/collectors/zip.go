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

func generateKey(fileName string) string{
	return ("hash/BID/1/"+os.Getenv("CONTAINER_NAME")+"/"+os.Getenv("COLLECTOR_NAME")+"/"+os.Getenv("DATA_NAME")+"/"+fileName)
}

func gzipFile(fileName string) {
	cmd := exec.Command("gzip", fileName)
	err := cmd.Start()
	cmd.Wait()
	if err != nil {
		panic(err)
		}
	}

func storeOnMinio(fileName string, bucket string, key string) {
	config := minio.Config{
		AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("MINIO_SECRET_ACCESS_KEY"),
		Endpoint:        os.Getenv("MINIO_HOST"),
		}
		s3Client, err := minio.New(config)
	    if err != nil {
	        log.Fatalln(err)
	    }  
	    object, err := os.Open(fileName)
		if err != nil {
			log.Fatalln(err)
		}
		defer object.Close()
		objectInfo, err := object.Stat()
		if err != nil {
			object.Close()
			log.Fatalln(err)
		}
		err = s3Client.PutObject(bucket, key, "application/octet-stream", objectInfo.Size(), object)
		if err != nil {
			log.Fatalln(err)
		}
	}
 
func backupHandler(w http.ResponseWriter, r *http.Request) {
    ev := os.Getenv("TO_ZIP")
    paths := strings.Split(ev, ":")
    for _, each := range paths {
        fmt.Fprintf(w, "Trying to zip %s\n", each)   
	    gzipFile(each)
	    storeOnMinio(each+".gz", "runs", generateKey(each+".gz"))
	}
	
	fmt.Fprintf(w, "SUCCESS")
}
 
func main() {
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}
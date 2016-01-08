package minio

import (
	"github.com/minio/minio-go"
	"os"
	"os/exec"
	"log"
	//"fmt"
)

func StoreOnMinio(fileName string, bucket string, key string) {
		s3Client, err := minio.New("195.176.181.55:9000", "CYNQML6R7V12MTT32W6P", "SQ96V5pg02Z3kZ/0ViF9YY6GwWzZvoBmElpzEEjn", true)
		if err != nil {
			log.Fatalln(err)
		}
		object, err := os.Open(fileName)
		if err != nil {
			log.Fatalln(err)
		}
		defer object.Close()
	
		st, _ := object.Stat()
		n, err := s3Client.PutObject(bucket, key, object, st.Size(), "application/octet-stream")
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("Uploaded", "my-objectname", " of size: ", n, "Successfully.")
	}

func GenerateKey(fileName string) string{
	// TODO: Use the hash library to generate a hash
	return ("hash/BID/1/"+os.Getenv("CONTAINER_NAME")+"/"+os.Getenv("COLLECTOR_NAME")+"/"+os.Getenv("DATA_NAME")+"/"+fileName)
}

func GzipFile(fileName string) {
	cmd := exec.Command("gzip", fileName)
	err := cmd.Start()
	cmd.Wait()
	if err != nil {
		panic(err)
		}
	}


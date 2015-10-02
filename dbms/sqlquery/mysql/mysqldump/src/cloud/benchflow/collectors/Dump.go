package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "github.com/minio/minio-go"
    "log"
)

func backupHandler(w http.ResponseWriter, r *http.Request) {
    cmd := exec.Command("mysqldump", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("DB_PORT_3306_TCP_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "--databases", os.Getenv("MYSQL_DB_NAME"))
    outfile, err := os.Create("./backup.sql")
    if err != nil {
        fmt.Fprintf(w, "ERROR:  %s", err)
        panic(err)
    }
    defer outfile.Close()
    cmd.Stdout = outfile
    err = cmd.Start()
    cmd.Wait()
    if err != nil {
        fmt.Fprintf(w, "ERROR:  %s", err)
        panic(err)
    }
    cmd = exec.Command("7zr", "a", "backup.7z", "backup.sql")
    err = cmd.Start()
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
    object, err := os.Open("backup.7z")
	if err != nil {
		log.Fatalln(err)
	}
	defer object.Close()
	objectInfo, err := object.Stat()
	if err != nil {
		object.Close()
		log.Fatalln(err)
	}
	err = s3Client.PutObject("backup", "backup.7z", "application/octet-stream", objectInfo.Size(), object)
	if err != nil {
		log.Fatalln(err)
	}
	for object := range s3Client.ListObjects("backup", "", true) {
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
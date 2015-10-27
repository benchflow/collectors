package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "github.com/minio/minio-go"
    "log"
)

var runCounter int

func backupHandler(w http.ResponseWriter, r *http.Request) {
	/*
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
    */
    
    cmd := exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("DB_PORT_3306_TCP_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; SELECT * FROM "+os.Getenv("TABLE_NAME")+";")
    cmd2 := exec.Command("sed", "s/\\t/\",\"/g;s/^/\"/;s/$/\"/;s/\\n//g")
    outfile, err := os.Create("./backup.csv")
    if err != nil {
        fmt.Fprintf(w, "ERROR:  %s", err)
        panic(err)
    }
    defer outfile.Close()
    cmd2.Stdin, _ = cmd.StdoutPipe()
    cmd2.Stdout = outfile
    err = cmd2.Start()
    cmd.Run()
    cmd2.Wait()
    if err != nil {
        fmt.Fprintf(w, "ERROR:  %s", err)
        panic(err)
    }
    
    cmd = exec.Command("gzip", "backup.csv")
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
	
	object, err := os.Open("backup.sql.gz")
	if err != nil {
		log.Fatalln(err)
	}
	defer object.Close()
	objectInfo, err := object.Stat()
	if err != nil {
		object.Close()
		log.Fatalln(err)
	}
	err = s3Client.PutObject("benchmarks", os.Getenv("CONTAINER_NAME")+"_mysqldump.sql.gz", "application/octet-stream", objectInfo.Size(), object)
	runCounter += 1
	if err != nil {
		log.Fatalln(err)
	}
	
	fmt.Fprintf(w, "SUCCESS")
}
 
func main() {
	runCounter = 1
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}
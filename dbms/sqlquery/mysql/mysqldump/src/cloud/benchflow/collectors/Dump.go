package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "github.com/minio/minio-go"
    "log"
    "strings"
)

func backupHandler(w http.ResponseWriter, r *http.Request) {
	
	// Minio client setup
	config := minio.Config{
	        AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY_ID"),
	        SecretAccessKey: os.Getenv("MINIO_SECRET_ACCESS_KEY"),
	        Endpoint:        os.Getenv("MINIO_HOST"),
	    }
	s3Client, err := minio.New(config)
	if err != nil {
		log.Fatalln(err)
	    }
	
	// Dumping the database as .sql
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
    
    cmd = exec.Command("gzip", "backup.sql")
	err = cmd.Start()
	cmd.Wait()
	if err != nil {
	    fmt.Fprintf(w, "ERROR:  %s", err)
	    panic(err)
	    }
	
	// Save to Minio
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
	err = s3Client.PutObject("benchmarks", "runs/a/"+"/"+os.Getenv("CONTAINER_NAME")+"_mysqldump_sql.gz", "application/octet-stream", objectInfo.Size(), object)
	if err != nil {
		log.Fatalln(err)
		}	
	
	// Retrieve table names
	ev := os.Getenv("TABLE_NAMES")
    tables := strings.Split(ev, ":")
    
    // Save the tables
    for _, each := range tables {
	    cmd := exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("DB_PORT_3306_TCP_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; SELECT * FROM "+each+";")
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
		
		object, err := os.Open("backup.csv.gz")
		if err != nil {
			log.Fatalln(err)
		}
		defer object.Close()
		objectInfo, err := object.Stat()
		if err != nil {
			object.Close()
			log.Fatalln(err)
		}
		err = s3Client.PutObject("benchmarks", "runs/a/"+"/"+os.Getenv("CONTAINER_NAME")+"_mysqldump_"+each+".csv.gz", "application/octet-stream", objectInfo.Size(), object)
		if err != nil {
			log.Fatalln(err)
		}
	}
    
    // Save the column types of the tables
    for _, each := range tables {
	    cmd := exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("DB_PORT_3306_TCP_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; SHOW FIELDS FROM "+each+";")
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
		
		object, err := os.Open("backup.csv.gz")
		if err != nil {
			log.Fatalln(err)
		}
		defer object.Close()
		objectInfo, err := object.Stat()
		if err != nil {
			object.Close()
			log.Fatalln(err)
		}
		err = s3Client.PutObject("benchmarks", "runs/a/"+"/"+os.Getenv("CONTAINER_NAME")+"_mysqldump_"+each+"_schema.csv.gz", "application/octet-stream", objectInfo.Size(), object)
		if err != nil {
			log.Fatalln(err)
		}	
	}
    
	fmt.Fprintf(w, "SUCCESS")
}
 
func main() {
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}
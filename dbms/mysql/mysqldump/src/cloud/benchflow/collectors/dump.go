package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "github.com/benchflow/commons/minio"
    "strings"
)

func backupHandler(w http.ResponseWriter, r *http.Request) {
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
	    cmd2.Stdin, _ = cmd.StdoutPipe()
	    cmd2.Stdout = outfile
	    err = cmd2.Start()
	    cmd.Run()
	    cmd2.Wait()
	    if err != nil {
	        fmt.Fprintf(w, "ERROR:  %s", err)
	        panic(err)
	    }
	    outfile.Close()
	    minio.GzipFile("backup.csv")
		minio.StoreOnMinio("backup.csv.gz", "runs", minio.GenerateKey(each+".csv.gz"))
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
	    cmd2.Stdin, _ = cmd.StdoutPipe()
	    cmd2.Stdout = outfile
	    err = cmd2.Start()
	    cmd.Run()
	    cmd2.Wait()
	    if err != nil {
	        fmt.Fprintf(w, "ERROR:  %s", err)
	        panic(err)
	    }
	    outfile.Close()
	    minio.GzipFile("backup.csv")
		minio.StoreOnMinio("backup.csv.gz", "runs", minio.GenerateKey(each+"_schema.csv.gz"))
	}
    
	fmt.Fprintf(w, "SUCCESS")
}
 
func main() {
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}

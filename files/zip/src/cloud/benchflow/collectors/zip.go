package main
 
import (
    "fmt"
    "net/http"
    "os"
    "strings"
    "github.com/Cerfoglg/commons/src/minio"
)
 
func backupHandler(w http.ResponseWriter, r *http.Request) {
    ev := os.Getenv("TO_ZIP")
    paths := strings.Split(ev, ":")
    for _, each := range paths {
        fmt.Fprintf(w, "Trying to zip %s\n", each)   
	    minio.GzipFile(each)
	    minio.StoreOnMinio(each+".gz", "runs", minio.GenerateKey(each+".gz"))
	}
	
	fmt.Fprintf(w, "SUCCESS")
}
 
func main() {
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}
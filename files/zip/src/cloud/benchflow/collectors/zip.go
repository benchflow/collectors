package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "strings"
)
 
func helloHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Greetings!")
}

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
    if err == nil {
        fmt.Fprintf(w, "Compressed")
    }
}
 
func main() {
    http.HandleFunc("/", helloHandler)
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}
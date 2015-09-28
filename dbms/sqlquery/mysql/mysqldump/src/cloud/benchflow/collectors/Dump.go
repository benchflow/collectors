package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
)
 
func helloHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Greetings!")
}

func backupHandler(w http.ResponseWriter, r *http.Request) {
    cmd := exec.Command("mysqldump", "-h", "db", "-P", "3306", "-u", "root", "-pPASSWORD", "--databases", "stuff")
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
    if err == nil {
        fmt.Fprintf(w, "SUCCESS")
    }
}
 
func main() {
    http.HandleFunc("/", helloHandler)
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}
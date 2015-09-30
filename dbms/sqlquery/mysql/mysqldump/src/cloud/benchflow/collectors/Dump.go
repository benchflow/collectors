package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
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
    cmd = exec.Command("./mc", "cp", "backup.7z", os.Getenv("MINIO_HOST"))
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
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}
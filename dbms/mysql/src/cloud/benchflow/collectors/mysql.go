package main
 
import (
    "net/http"
    "os"
    "os/exec"
    "github.com/benchflow/commons/minio"
    "github.com/benchflow/commons/kafka"
    "strings"
    "encoding/json"
)

type Response struct {
  Successful bool
}

func backupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		w.WriteHeader(405)
		return	
	}
	// Generating key for Minio
	databaseMinioKey := minio.GenerateKey(os.Getenv("MYSQL_DB_NAME"), os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
	
	// Save whole database as mysqldump
	cmd := exec.Command("mysqldump", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("MYSQL_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), os.Getenv("MYSQL_DB_NAME"))
	outfile, err := os.Create("/app/"+os.Getenv("MYSQL_DB_NAME")+"_backup.sql")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
	    return
    }
    cmd.Stdout = outfile
    cmd.Start()
    err = cmd.Wait()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
	    return
    }
    outfile.Close()
   
    minio.GzipFile("/app/"+os.Getenv("MYSQL_DB_NAME")+"_backup.sql")
    minio.SendGzipToMinio("/app/"+os.Getenv("MYSQL_DB_NAME")+"_backup.sql.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), databaseMinioKey+"/"+os.Getenv("MYSQL_DB_NAME")+"_backup.sql", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
   
	err = os.Remove("/app/"+os.Getenv("MYSQL_DB_NAME")+"_backup.sql.gz")
	if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
	    return
    }
	
	// Retrieve table names
	ev := os.Getenv("TABLE_NAMES")
    tables := strings.Split(ev, ",")
    
    // Save Table sizes
    cmd = exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("MYSQL_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; select table_schema AS Db, sum(data_length+index_length) AS Bytes from information_schema.tables where table_schema='"+os.Getenv("MYSQL_DB_NAME")+"' group by 1;")
    cmd2 := exec.Command("sed", "s/\\t/\",\"/g;s/^/\"/;s/$/\"/;s/\\n//g")
    outfile, err = os.Create("/app/database_table_sizes_backup.csv")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
	    return
    }
    cmd2.Stdin, _ = cmd.StdoutPipe()
    cmd2.Stdout = outfile
    err = cmd2.Start()
    cmd.Run()
    cmd2.Wait()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
	    return
    }
    outfile.Close()
    minio.GzipFile("/app/database_table_sizes_backup.csv")
    minio.SendGzipToMinio("/app/database_table_sizes_backup.csv.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), databaseMinioKey+"/database_table_sizes.csv.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
	err = os.Remove("/app/database_table_sizes_backup.csv.gz")
	if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
	    return
    }
	
    
    // Save the tables
    for _, each := range tables {
	    cmd := exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("MYSQL_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; SELECT * FROM "+each+";")
	    cmd2 := exec.Command("sed", "s/\\t/\",\"/g;s/^/\"/;s/$/\"/;s/\\n//g")
	    outfile, err := os.Create("/app/"+each+"_backup.csv")
	    if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }
	    cmd2.Stdin, _ = cmd.StdoutPipe()
	    cmd2.Stdout = outfile
	    err = cmd2.Start()
	    cmd.Run()
	    cmd2.Wait()
	    if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }
	    outfile.Close()
	    minio.GzipFile("/app/"+each+"_backup.csv")
	    minio.SendGzipToMinio("/app/"+each+"_backup.csv.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), databaseMinioKey+"/"+each+".csv.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		err = os.Remove("/app/"+each+"_backup.csv.gz")
		if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }
	}
    
    // Save the column types of the tables
    for _, each := range tables {
	    cmd := exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("MYSQL_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; SHOW FIELDS FROM "+each+";")
	    cmd2 := exec.Command("sed", "s/\\t/\",\"/g;s/^/\"/;s/$/\"/;s/\\n//g")
	    outfile, err := os.Create("/app/"+each+"_backup_schema.csv")
	    if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }
	    cmd2.Stdin, _ = cmd.StdoutPipe()
	    cmd2.Stdout = outfile
	    err = cmd2.Start()
	    cmd.Run()
	    cmd2.Wait()
	    if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }
	    outfile.Close()
	    minio.GzipFile("/app/"+each+"_backup_schema.csv")
	    minio.SendGzipToMinio("/app/"+each+"_backup_schema.csv.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), databaseMinioKey+"/"+each+"_schema.csv.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		err = os.Remove("/app/"+each+"_backup_schema.csv.gz")
		if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }
	}
    kafka.SignalOnKafka(databaseMinioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), "mysql", "mysql", "host", os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	
	response := Response{true}
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	    return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
}
 
func main() {
    http.HandleFunc("/store", backupHandler)
    http.ListenAndServe(":8080", nil)
}

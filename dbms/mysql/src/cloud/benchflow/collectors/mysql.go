package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "bytes"
    "github.com/benchflow/commons/minio"
    "github.com/Shopify/sarama"
    "log"
    "encoding/json"
    "strings"
    "strconv"
)

type KafkaMessage struct {
	SUT_name string `json:"SUT_name"`
	SUT_version string `json:"SUT_version"`
	Minio_key string `json:"minio_key"`
	Trial_id string `json:"trial_id"`
	Experiment_id string `json:"experiment_id"`
	Total_trials_num int `json:"total_trials_num"`
	Collector_name string `json:"collector_name"`
	}

func signalOnKafka(databaseMinioKey string) {
	totalTrials, _ := strconv.Atoi(os.Getenv("TOTAL_TRIALS_NUM"))
	kafkaMsg := KafkaMessage{SUT_name: os.Getenv("SUT_NAME"), SUT_version: os.Getenv("SUT_VERSION"), Minio_key: databaseMinioKey, Trial_id: os.Getenv("TRIAL_ID"), Experiment_id: os.Getenv("EXPERIMENT_ID"), Total_trials_num: totalTrials, Collector_name: os.Getenv("COLLECTOR_NAME")}
	jsMessage, err := json.Marshal(kafkaMsg)
	if err != nil {
		log.Printf("Failed to marshall json message")
		}
	//TODO: the kafka host should be passed as an environment variable
	producer, err := sarama.NewSyncProducer([]string{os.Getenv("KAFKA_HOST")+":9092"}, nil)
	if err != nil {
	    log.Fatalln(err)
	}
	defer func() {
	    if err := producer.Close(); err != nil {
	        log.Fatalln(err)
	    }
	}()
	msg := &sarama.ProducerMessage{Topic: os.Getenv("KAFKA_TOPIC"), Value: sarama.StringEncoder(jsMessage)}
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
	    log.Printf("FAILED to send message: %s\n", err)
	    } else {
	    log.Printf("> message sent to partition %d at offset %d\n", partition, offset)
	    }
	}

func backupHandler(w http.ResponseWriter, r *http.Request) {
	// Generating key for Minio
	databaseMinioKey := minio.GenerateKey(os.Getenv("MYSQL_DB_NAME"))

	log.Printf("Minio Key: " + databaseMinioKey)
	
	// Retrieve table names
	ev := os.Getenv("TABLE_NAMES")
    tables := strings.Split(ev, ",")
    
    // cmdd := exec.Command("touch", "/app/backup.csv")
    // cmdd.Run()
    // cmdd.Wait()
    // cmdd = exec.Command("chmod", "777", "/app/backup.csv")
    // cmdd.Run()
    // cmdd.Wait()
    /*
    for _,each := range tables {
		cmd := exec.Command("mysqldump", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("MYSQL_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), os.Getenv("MYSQL_DB_NAME"), each)
		outfile, err := os.Create("/app/"+each+"_backup.sql")
	    if err != nil {
	        fmt.Fprintf(w, "ERROR:  %s", err)
	        panic(err)
	    }
	    cmd.Stdout = outfile
	    err = cmd.Start()
	    cmd.Wait()
	    if err != nil {
	        fmt.Fprintf(w, "ERROR:  %s", err)
	        panic(err)
	        }
	    outfile.Close()
	    minio.GzipFile("/app/"+each+"_backup.sql")
		callMinioClient("/app/"+each+"_backup.sql.gz", os.Getenv("MINIO_ALIAS"), databaseMinioKey+"/"+each+".sql.gz")
		err = os.Remove("/app/"+each+"_backup.sql.gz")
		if err != nil {
	        fmt.Fprintf(w, "ERROR:  %s", err)
	        panic(err)
	    }
	}
	*/
    
    // Save Table sizes
    cmd := exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("MYSQL_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; select table_schema AS Db, sum(data_length+index_length) AS Bytes from information_schema.tables where table_schema='"+os.Getenv("MYSQL_DB_NAME")+"' group by 1;")
    cmd2 := exec.Command("sed", "s/\\t/\",\"/g;s/^/\"/;s/$/\"/;s/\\n//g")
    outfile, err := os.Create("/app/database_table_sizes_backup.csv")
    // outfile, err := os.Open("/app/backup.csv")
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
    minio.GzipFile("/app/database_table_sizes_backup.csv")
    callMinioClient("/app/database_table_sizes_backup.csv.gz", os.Getenv("MINIO_ALIAS"), databaseMinioKey+"/database_table_sizes.csv.gz")
	//minio.StoreOnMinio("backup.csv.gz", "runs", databaseMinioKey+each+".csv.gz")
	err = os.Remove("/app/database_table_sizes_backup.csv.gz")
	if err != nil {
        fmt.Fprintf(w, "ERROR:  %s", err)
        panic(err)
    }
	
    
    // Save the tables
    for _, each := range tables {
	    cmd := exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("MYSQL_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; SELECT * FROM "+each+";")
	    cmd2 := exec.Command("sed", "s/\\t/\",\"/g;s/^/\"/;s/$/\"/;s/\\n//g")
	    outfile, err := os.Create("/app/"+each+"_backup.csv")
	    // outfile, err := os.Open("/app/backup.csv")
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
	    minio.GzipFile("/app/"+each+"_backup.csv")
	    callMinioClient("/app/"+each+"_backup.csv.gz", os.Getenv("MINIO_ALIAS"), databaseMinioKey+"/"+each+".csv.gz")
		//minio.StoreOnMinio("backup.csv.gz", "runs", databaseMinioKey+each+".csv.gz")
		err = os.Remove("/app/"+each+"_backup.csv.gz")
		if err != nil {
	        fmt.Fprintf(w, "ERROR:  %s", err)
	        panic(err)
	    }
	}
    
    // cmdd = exec.Command("touch", "/app/backup.csv")
    // cmdd.Run()
    // cmdd.Wait()
    // cmdd = exec.Command("chmod", "777", "/app/backup.csv")
    // cmdd.Run()
    // cmdd.Wait()
    
    // Save the column types of the tables
    for _, each := range tables {
	    cmd := exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("MYSQL_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; SHOW FIELDS FROM "+each+";")
	    cmd2 := exec.Command("sed", "s/\\t/\",\"/g;s/^/\"/;s/$/\"/;s/\\n//g")
	    outfile, err := os.Create("/app/"+each+"_backup_schema.csv")
	    // outfile, err := os.Open("/app/backup.csv")
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
	    minio.GzipFile("/app/"+each+"_backup_schema.csv")
	    callMinioClient("/app/"+each+"_backup_schema.csv.gz", os.Getenv("MINIO_ALIAS"), databaseMinioKey+"/"+each+"_schema.csv.gz")
		//minio.StoreOnMinio("backup.csv.gz", "runs", databaseMinioKey+each+"_schema.csv.gz")
		err = os.Remove("/app/"+each+"_backup_schema.csv.gz")
		if err != nil {
	        fmt.Fprintf(w, "ERROR:  %s", err)
	        panic(err)
	    }
	}
    signalOnKafka(databaseMinioKey)
	fmt.Fprintf(w, "SUCCESS")
}

func callMinioClient(fileName string, minioHost string, minioKey string) {
		//TODO: change, we are using sudo to elevate the priviledge in the container, but it is not nice
		//NOTE: it seems that the commands that are not in PATH, should be launched using sh -c
		log.Printf("sh -c sudo /app/mc --quiet cp " + fileName + " " + minioHost + "/runs/" + minioKey)
		cmd := exec.Command("sh", "-c", "sudo /app/mc --quiet cp " + fileName + " " + minioHost + "/runs/" + minioKey)
    	var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
		    fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		    return
		}
		fmt.Println("Result: " + out.String())
}
 
func main() {
    http.HandleFunc("/data", backupHandler)
    http.ListenAndServe(":8080", nil)
}

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
)

type KafkaMessage struct {
	SUT_name string `json:"SUT_name"`
	SUT_version string `json:"SUT_version"`
	Minio_key string `json:"minio_key"`
	Trial_id string `json:"trial_id"`
	Experiment_id string `json:"experiment_id"`
	Total_trials_num int `json:"total_trials_num"`
	}

func signalOnKafka(databaseMinioKey string) {
	kafkaMsg := KafkaMessage{SUT_name: "Camunda", SUT_version: "", Minio_key: databaseMinioKey, Trial_id: os.Getenv("TRIAL_ID"), Experiment_id: os.Getenv("EXPERIMENT_ID"), Total_trials_num: os.Getenv("TOTAL_TRIALS_NUM")}
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
	msg := &sarama.ProducerMessage{Topic: os.Getenv("COLLECTOR_NAME"), Value: sarama.StringEncoder(jsMessage)}
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
    tables := strings.Split(ev, ":")
    
    
    // cmdd := exec.Command("touch", "/app/backup.csv")
    // cmdd.Run()
    // cmdd.Wait()
    // cmdd = exec.Command("chmod", "777", "/app/backup.csv")
    // cmdd.Run()
    // cmdd.Wait()
    
    // Save the tables
    for _, each := range tables {
	    cmd := exec.Command("mysql", "-h", os.Getenv("MYSQL_HOST"), "-P", os.Getenv("MYSQL_PORT"), "-u", os.Getenv("MYSQL_USER"), "-p" + os.Getenv("MYSQL_USER_PASSWORD"), "-e", "USE "+os.Getenv("MYSQL_DB_NAME")+"; SELECT * FROM "+each+";")
	    cmd2 := exec.Command("sed", "s/\\t/\",\"/g;s/^/\"/;s/$/\"/;s/\\n//g")
	    outfile, err := os.Create("/app/backup.csv")
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
	    minio.GzipFile("/app/backup.csv")
	    callMinioClient("/app/backup.csv.gz", os.Getenv("MINIO_ALIAS"), databaseMinioKey+"/"+each+".csv.gz")
		//minio.StoreOnMinio("backup.csv.gz", "runs", databaseMinioKey+each+".csv.gz")
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
	    outfile, err := os.Create("/app/backup_schema.csv")
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
	    minio.GzipFile("/app/backup_schema.csv")
	    callMinioClient("/app/backup_schema.csv.gz", os.Getenv("MINIO_ALIAS"), databaseMinioKey+"/"+each+"_schema.csv.gz")
		//minio.StoreOnMinio("backup.csv.gz", "runs", databaseMinioKey+each+"_schema.csv.gz")
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

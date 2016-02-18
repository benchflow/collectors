package main
 
import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "encoding/json"
    "log"
    "strings"
    "bytes"
    "github.com/benchflow/commons/minio"
    "github.com/Shopify/sarama"
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

func signalOnKafka(minioKey string) {
	totalTrials := strconv.Atoi(os.Getenv("TOTAL_TRIALS_NUM"))
	kafkaMsg := KafkaMessage{SUT_name: "Camunda", SUT_version: "", Minio_key: minioKey, Trial_id: os.Getenv("TRIAL_ID"), Experiment_id: os.Getenv("EXPERIMENT_ID"), Total_trials_num: totalTrials}
	jsMessage, err := json.Marshal(kafkaMsg)
	if err != nil {
		log.Printf("Failed to marshall json message")
		}
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
    ev := os.Getenv("TO_ZIP")
    paths := strings.Split(ev, ":")
    for _, each := range paths {
        fmt.Fprintf(w, "Trying to zip %s\n", each)
        folderList := strings.Split(each, "/")
	    folderName := folderList[len(folderList)-1]
        cmd := exec.Command("tar", "-zcvf", folderName+".tar.gz", each)
		err := cmd.Start()
		cmd.Wait()
		if err != nil {
			panic(err)
			}  
	    minioKey := minio.GenerateKey(folderName+".tar.gz")
		callMinioClient(folderName+".tar.gz", os.Getenv("MINIO_ALIAS"), minioKey)
	    //minio.StoreOnMinio(folderName+".tar.gz", "runs", minioKey)
	    signalOnKafka(minioKey)
	}
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
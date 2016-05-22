package main
 
import (
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "os/exec"
    "encoding/json"
    "log"
    "bytes"
    "github.com/benchflow/commons/minio"
    "github.com/Shopify/sarama"
    "strconv"
)

var fabanOutputRoot = "/app/faban/output/"

type KafkaMessage struct {
	SUT_name string `json:"SUT_name"`
	SUT_version string `json:"SUT_version"`
	Minio_key string `json:"minio_key"`
	Trial_id string `json:"trial_id"`
	Experiment_id string `json:"experiment_id"`
	Container_id string `json:"container_id"`
	Total_trials_num int `json:"total_trials_num"`
	Collector_name string `json:"collector_name"`
	}

func signalOnKafka(minioKey string, containerID string) {
	totalTrials, _ := strconv.Atoi(os.Getenv("BENCHFLOW_TRIAL_TOTAL_NUM"))
	kafkaMsg := KafkaMessage{SUT_name: os.Getenv("SUT_NAME"), SUT_version: os.Getenv("SUT_VERSION"), Minio_key: minioKey, Trial_id: os.Getenv("BENCHFLOW_TRIAL_ID"), Experiment_id: os.Getenv("BENCHFLOW_EXPERIMENT_ID"), Container_id: containerID, Total_trials_num: totalTrials, Collector_name: os.Getenv("BENCHFLOW_COLLECTOR_NAME")}
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
	msg := &sarama.ProducerMessage{Topic: os.Getenv("KAFKA_TOPIC"), Value: sarama.StringEncoder(jsMessage)}
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
	    log.Printf("FAILED to send message: %s\n", err)
	    } else {
	    log.Printf("> message sent to partition %d at offset %d\n", partition, offset)
	    }
	}
 
func backupHandler(w http.ResponseWriter, r *http.Request) {
    fabanRunId := r.FormValue("faban_run_id")
    path := fabanOutputRoot+fabanRunId+"/"
    files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	paths := []string{}
    for _, file := range files {
		paths = append(paths, path+"/"+file.Name())
	}
	cmd := exec.Command("gzip", paths...)
	err = cmd.Start()
	cmd.Wait()
	if err != nil {
		panic(err)
		} 
	minioKey := minio.GenerateKey(fabanRunId)
    for _, each := range files { 
		callMinioClient(path+"/"+each.Name()+".gz", os.Getenv("MINIO_ALIAS"), minioKey+"/"+each.Name()+".gz")
	}
    signalOnKafka(minioKey, "faban")
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
    http.HandleFunc("/store", backupHandler)
    http.ListenAndServe(":8080", nil)
}
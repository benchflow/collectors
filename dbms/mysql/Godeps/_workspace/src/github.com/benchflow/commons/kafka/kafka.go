package kafka

import (
    "github.com/Shopify/sarama"
    "log"
    "encoding/json"
)

type KafkaMessage struct {
	Minio_key string `json:"minio_key"`
	Trial_id string `json:"trial_id"`
	Experiment_id string `json:"experiment_id"`
	Container_id string `json:"container_id"`
	Host_id string `json:"host_id"`
	Collector_name string `json:"collector_name"`
	}

func SignalOnKafka(minioKey string, trialID string, experimentID string, containerID string, hostID string, collectorName string, kafkaHost string, kafkaPort string, kafkaTopic string) {
	kafkaMsg := KafkaMessage{Minio_key: minioKey, Trial_id: trialID, Experiment_id: experimentID, Container_id: containerID, Host_id: hostID, Collector_name: collectorName}
	jsMessage, err := json.Marshal(kafkaMsg)
	if err != nil {
		log.Printf("Failed to marshall json message")
		}
	producer, err := sarama.NewSyncProducer([]string{kafkaHost+":"+kafkaPort}, nil)
	if err != nil {
	    log.Fatalln(err)
	}
	defer func() {
	    if err := producer.Close(); err != nil {
	        log.Fatalln(err)
	    }
	}()
	msg := &sarama.ProducerMessage{Topic: kafkaTopic, Value: sarama.StringEncoder(jsMessage)}
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
	    log.Printf("FAILED to send message: %s\n", err)
	    } else {
	    log.Printf("> message sent to partition %d at offset %d\n", partition, offset)
	    }
	}
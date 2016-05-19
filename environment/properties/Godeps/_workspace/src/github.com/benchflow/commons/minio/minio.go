package minio

import (
	"os"
	"os/exec"
	"strings"
)

func GenerateKey(fileName string) string {
	// TODO: Use the hash library to generate a hash
	trialId := os.Getenv("TRIAL_ID")
	t := strings.Split(trialId, "_")
	return ("hash/"+t[0]+"/"+t[1]+"/"+os.Getenv("CONTAINER_NAME")+"/"+os.Getenv("COLLECTOR_NAME")+"/"+os.Getenv("DATA_NAME")+"/"+fileName)
}

func GzipFile(fileName string) {
	cmd := exec.Command("gzip", fileName)
	err := cmd.Start()
	cmd.Wait()
	if err != nil {
		panic(err)
		}
}


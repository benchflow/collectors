package main

import (
	"fmt"
	"net"
	"time"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"github.com/fsouza/go-dockerclient"
	"github.com/benchflow/commons/minio"
	"github.com/benchflow/commons/kafka"
	"encoding/json"
)

//Structure of the response message for the API requests
type Response struct {
  Status string
  Message string
}

//Function to marshall and write the response to the API requests
func writeJSONResponse(w http.ResponseWriter, status string, message string) {
	response := Response{status, message}
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println(err)
	    return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(js)	
}

//Structure representing a container and defining the channel for retrieving stats over time from the Docker client
type Container struct {
	Name         string
	ID           string
	statsChannel chan *docker.Stats
	doneChannel  chan bool
	Network      string
}

//List of the containers to observe
var containers []Container
//Host ID
var hostID string
//Channel to receive signal to stop collecting
var stopChannel chan bool
//Waitgroup object to await termination of the collections from all containers
var waitGroup sync.WaitGroup
//Var defining whether a collection is being performed or not
var collecting bool

//Function to attach to a container to start collecting stats
func attachToContainer(client docker.Client, container Container) {
	go func() {
		_ = client.Stats(docker.StatsOptions{
			ID:      container.ID,
			Stats:   container.statsChannel,
			Stream:  true,
			Done:    container.doneChannel,
			Timeout: 0,
		})
	}()
}

//Function to collect the stats, will continue to do so till stopChannel is signalled to stop
func collectStats(container Container) {
	go func() {
		//Create temp file for the stats and variable to contain the JSON objects containing the stats at a given moment
		var e docker.Env
		fo, err := os.Create("/app/"+container.Name+"_stats_tmp")
		if err != nil {
	        panic(err)
	    }
		for true {
			select {
			case <- stopChannel:
				//If signalled to stop, stop collecting
				close(container.doneChannel)
				fo.Close()
				waitGroup.Done()
				return
			default:
				//If not signalled to stop, collect stats data and write it to the temp file
				dat := (<-container.statsChannel)
				e.SetJSON("dat", dat)
				fo.Write([]byte(e.Get("dat")))
				fo.Write([]byte("\n"))
				}
		}
	}()
}

//Function to collect the network stats using nethogs if net==host on the container
func collectNetworkStats(container Container, client docker.Client) {
	go func() {
		//Create temp files
		foNet, err := os.Create("/app/"+container.Name+"_network_tmp")
		if err != nil {
	        panic(err)
	    }
		foTop, err := os.Create("/app/"+container.Name+"_top_tmp")
		if err != nil {
	        panic(err)
	    }
		//Get the list of network interfaces
		interfaces, err := net.Interfaces()
		if err != nil {
	        panic(err)
	    }
		//Set nethogs options
		var nethogsOptions []string
		nethogsOptions = append(nethogsOptions, "-t")

		var interfaceNames []string
		//Compute interfaces that can and should be monitored
		//This fixes: http://askubuntu.com/questions/261024/nethogs-ioctl-failed-while-establishing-local-ip
		//TODO: investigate possible improvements
	    for _, each := range interfaces {
	    	  //Reference: http://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
	    	  //TODO: investigate possible not working with IPV6
	          addrs, _ := each.Addrs()
	          
	          atLeastOneToMonitor := false

	          for _, addr := range addrs {
	             var ip net.IP
	             switch v := addr.(type) {
	             case *net.IPNet:
	                     // fmt.Println("IPNet");
	                     ip = v.IP
	             case *net.IPAddr:
	                     // fmt.Println("IPAddr");
	                     ip = v.IP
	             }
	             // if the ip is a GlobalUnicast or the LoopBack one, we monitor it.
	             // We want to monitor network utilization across IPs
	             // As for example, Docker virtual subnet (https://docs.docker.com/v1.7/articles/networking/) it is not monitored
	             if(ip.IsGlobalUnicast() || ip.IsLoopback()){
	               atLeastOneToMonitor = true
	               break
	             }
	         }

	         if(atLeastOneToMonitor){
	            interfaceNames = append(interfaceNames, each.Name)
	         }

	         // fmt.Println(interfaceNames);
	    }
		
		//Run nethogs to collect data
		nethogsOptions = append(nethogsOptions, interfaceNames...)
		cmd := exec.Command("/usr/sbin/nethogs", nethogsOptions...)
		cmd.Stdout = foNet
		err = cmd.Start()
	    if err != nil {
	        panic(err)
	    }
		for true {
			select {
			default:
				//If not signalled to stop, write nethogs data to file every 750 milliseconds
				top, err := client.TopContainer(container.Name, "")
				if err != nil {
		        	panic(err)
		    	}
				var e docker.Env
				e.SetJSON("top", top)
				foTop.WriteString(e.Get("top")+"\n")
				time.Sleep(750 * time.Millisecond)
			case <- stopChannel:
				//If signalled to stop, kill nethogs and stop collecting
				cmd.Process.Kill()
				foNet.Close()
				foTop.Close()
				waitGroup.Done()
				return
			}
		}
	}()
}

//Function to create the Docker client object to communicate with Docker, using the Docker socket (shared with the container)
func createDockerClient() docker.Client {
	//path := os.Getenv("DOCKER_CERT_PATH")
	//endpoint := "tcp://192.168.99.100:2376"
    //ca := fmt.Sprintf("%s/ca.pem", path)
    //cert := fmt.Sprintf("%s/cert.pem", path)
    //key := fmt.Sprintf("%s/key.pem", path)
    //client, err := docker.NewTLSClient(endpoint, cert, key, ca)
	endpoint := "unix:///var/run/docker.sock"
    client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	return *client
}

//Function to begin collecting when client requests on /start
func startCollecting(w http.ResponseWriter, r *http.Request) {
	//If request method is not POST then respond with method not allowed
	if r.Method != "POST" {
		w.WriteHeader(405)
		return	
	}
	
	//If already collecting respond with collection already in progress
	if collecting {
		writeJSONResponse(w, "FAILED", "Collection already in progress")
		fmt.Println("Collection already in progress")
		return
	}
	
	//Create Docker client
	client := createDockerClient()
	
	//Get host ID
	hostInfo, err := client.Info()
	if err != nil {
		panic(err)
	}
	hostID = hostInfo.ID
	
	//Get list of containers to observe
	contEV := os.Getenv("CONTAINERS")
	conts := strings.Split(contEV, ",")
	containers = []Container{}
	//Create stop channel
	stopChannel = make(chan bool)
	//For every container to observe, start the collection
	for _, each := range conts {
		containerInspect, err := client.InspectContainer(each)
		networks := containerInspect.NetworkSettings.Networks
		// Assuming bridge by default if not host
		/*
		Possible values of a network:
		--net="bridge"          Connect a container to a network
                                'bridge': create a network stack on the default Docker bridge
                                'none': no networking
                                'container:<name|id>': reuse another container's network stack
                                'host': use the Docker host network stack
                                '<network-name>|<network-id>': connect to a user-defined network
		*/
		//Check if net is host or not
		network := "bridge"
		for k := range networks {
			if k == "host" {
				network = "host"
				}
			}
		//Container ID
		ID := containerInspect.ID
		if err != nil {
			panic(err)
			}
		//Start collecting
		statsChannel := make(chan *docker.Stats)
		doneChannel := make(chan bool)
		c := Container{Name: each, ID: ID, statsChannel: statsChannel, doneChannel: doneChannel, Network: network}
		containers = append(containers, c)
		attachToContainer(client, c)
		collectStats(c)
		waitGroup.Add(1)
		//If net is host, collect network stats with nethogs
		if(network == "host") {
			collectNetworkStats(c, client)
			waitGroup.Add(1)
		}
	}
	//Set that we are collecting 
	collecting = true
	
	//Respond to the client
	writeJSONResponse(w, "SUCCESS", "The collection was started successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
	fmt.Println("The collection was started successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
}

//Function to stop the collection when receiving a request on /stop
func stopCollecting(w http.ResponseWriter, r *http.Request) {
	//If request method is not PUT then respond with method not allowed
	if r.Method != "PUT" {
		w.WriteHeader(405)
		return	
	}
	//If not collecting, then respond that collection is not in progress right now
	if !collecting {
		writeJSONResponse(w, "FAILED", "No collection in progress")
		fmt.Println("No collection in progress")
		return
	}
	//Signal to stop by closing the stop channel
	close(stopChannel)
	//Wait till all collections have stopped
	waitGroup.Wait()
	//Set that we are done collecting
	collecting = false
	
	composedMinioKey := ""
	composedContainerIds := ""
	composedContainerNames := ""
	//For all containers observed, save the stats
	for _, container := range containers {
		//Gzip files
		minio.GzipFile("/app/"+container.Name+"_stats_tmp")
		if container.Network == "host" {
			minio.GzipFile("/app/"+container.Name+"_network_tmp")
			minio.GzipFile("/app/"+container.Name+"_top_tmp")
		}
		//Generate Minio key
		minioKey := minio.GenerateKey(container.Name, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), os.Getenv("BENCHFLOW_CONTAINER_NAME"), os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("BENCHFLOW_DATA_NAME"))
		composedMinioKey = composedMinioKey+minioKey+","
		composedContainerIds = composedContainerIds+container.ID+","
		cName := strings.Split(container.Name, "_")[0]
		composedContainerNames = composedContainerNames+cName+","
		//Send files to Minio
		minio.SendGzipToMinio("/app/"+container.Name+"_stats_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_stats.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		if container.Network == "host" {
			minio.SendGzipToMinio("/app/"+container.Name+"_network_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_network.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
			minio.SendGzipToMinio("/app/"+container.Name+"_top_tmp.gz", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT"), minioKey+"_top.gz", os.Getenv("MINIO_ACCESSKEYID"), os.Getenv("MINIO_SECRETACCESSKEY"))
		}
		//Remove temp files
		err := os.Remove("/app/"+container.Name+"_stats_tmp.gz")
		if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	        fmt.Println(err)
    		return
	    }
		if container.Network == "host" {
			err = os.Remove("/app/"+container.Name+"_network_tmp.gz")
			if err != nil {
		        http.Error(w, err.Error(), http.StatusInternalServerError)
		        fmt.Println(err)
	    		return
		    }
			err = os.Remove("/app/"+container.Name+"_top_tmp.gz")
			if err != nil {
		        http.Error(w, err.Error(), http.StatusInternalServerError)
		        fmt.Println(err)
	    		return
		    }
		}
	}
	//Signal to Kafka
	composedMinioKey = strings.TrimRight(composedMinioKey, ",")
	composedContainerIds = strings.TrimRight(composedContainerIds, ",")
	composedContainerNames = strings.TrimRight(composedContainerNames, ",")
	kafka.SignalOnKafka(composedMinioKey, os.Getenv("BENCHFLOW_TRIAL_ID"), os.Getenv("BENCHFLOW_EXPERIMENT_ID"), composedContainerIds, composedContainerNames, hostID, os.Getenv("BENCHFLOW_COLLECTOR_NAME"), os.Getenv("KAFKA_HOST"), os.Getenv("KAFKA_PORT"), os.Getenv("KAFKA_TOPIC"))
	
	//Respond to the client
	writeJSONResponse(w, "SUCCESS", "The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
	fmt.Println("The collection was performed successfully for "+os.Getenv("BENCHFLOW_TRIAL_ID"))
}

func main() {
	collecting = false
	
	http.HandleFunc("/start", startCollecting)
	http.HandleFunc("/stop", stopCollecting)
	http.ListenAndServe(":" + os.Getenv("EXPOSED_PORT"), nil)
}
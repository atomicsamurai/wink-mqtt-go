package main

import (
	"os"
	"os/exec"
	"strings"
	"bufio"
    "fmt"
    "log"
	"strconv"
	"encoding/json"
	"io/ioutil"
    // "time"
	// "database/sql"
    mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/fsnotify/fsnotify"
	// _ "modernc.org/sqlite"
)

var sqlquery string = "select d.masterId, s.attributeId, s.value_SET FROM zwaveDeviceState AS s,zwaveDevice AS d WHERE d.nodeId=s.nodeId AND s.attributeId IN (10,15,23)"
var mqttBrokerPort int = 1883

var devices map[int]string
var attributes map[int]string
var client mqtt.Client
var config Config

type NameID struct {
	Id int `json:"id"`
	Name string `json:"name"`
}

type Config struct {
	DbFile string `json:"dbfile"`
	MqttBroker string `json:"mqttbroker"`
	MqttUsername string `json:"mqttusername"`
	MqttPassword string `json:"mqttpassword"`
	SubscribeTopic string `json:"subscribeTopic"`
	PublishTopicTemplate string `json:"publishTopicTemplate"`
	AvailableTopic string `json:"availableTopic"`
	OnlinePayload string `json:"onlinePayload"`
	OfflinePayload string `json:"offlinePayload"`
	LogLevel int `json:"logLevel"`
	Devices []NameID `json:"devices"`
	Attributes []NameID `json:"attributes"`
}

var cached_state = map[string]map[string]string{}
// var newdata = map[string]map[string]string{}

func readConfig(filename string) {
	// jsonFile, err := os.Open("/opt/wink-mqtt/config.json")
	jsonFile, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully Opened config.json")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	// config := Config{}
	json.Unmarshal(byteValue, &config)
	// log.Println(config.Devices[0])
}

func getDeviceIDFromName(name string) string {
	for index, value := range config.Devices {
		_ = index
		if value.Name == name {
			return strconv.Itoa(value.Id)
		}
	}
	return ""
}

func getDeviceNameFromID(id string) string {
	for index, value := range config.Devices {
		_ = index
		if strconv.Itoa(value.Id) == id {
			return value.Name
		}
	}
	return ""
}

func getAttribIDFromName(attribute string) string {
	for index, value := range config.Attributes {
		_ = index
		if value.Name == attribute {
			return strconv.Itoa(value.Id)
		}
	}
	return ""
}

func getAttributeNameFromID(id string) string {
	for index, value := range config.Attributes {
		_ = index
		if strconv.Itoa(value.Id) == id {
			return value.Name
		}
	}
	return ""
}

func isNameRelevant(name string) bool {
	for index, value := range config.Devices {
		_ = index
		// log.Printf("name: %s, devicename: %s\n", name, value.Name)
		if value.Name == name {
			return true
		}
	}
	return false
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    log.Printf("messagePubHandler Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var messageSubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
	split_array := strings.Split(msg.Topic(), "/")
	// log.Printf("name: %s", split_array[1])
	if isNameRelevant(split_array[1]) {
		log.Printf("command for device %s\n", split_array[1])
		if split_array[3] == "set" {
			log.Printf("sending %s:%s to device %s\n", split_array[2], string(msg.Payload()[:]), split_array[1])
			updateDevice(split_array[1], split_array[2], string(msg.Payload()[:]))
		}
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
    log.Println("Connected")
	token := client.Publish(config.AvailableTopic, 0, true, config.OnlinePayload)
	token.Wait()
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
    log.Printf("Connect lost: %v\n", err)
}

func publishMQTT(client mqtt.Client, device string, attribute string, value string) {
	deviceName := getDeviceNameFromID(device)
	attributeName := getAttributeNameFromID(attribute)
	log.Println("publishing to", fmt.Sprintf(config.PublishTopicTemplate, deviceName, attributeName), value)
	token := client.Publish(fmt.Sprintf(config.PublishTopicTemplate, deviceName, attributeName), 0, true, value)
	token.Wait()
}

func subscribeMQTT(client mqtt.Client) {
    token := client.Subscribe(config.SubscribeTopic, 1, messageSubHandler)
    token.Wait()
	log.Printf("Subscribed to topic: %s\n", config.SubscribeTopic)
}

func initMQTT () {
    opts := mqtt.NewClientOptions()
    opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.MqttBroker, mqttBrokerPort))
    opts.SetClientID("go_mqtt_client")
    opts.SetUsername(config.MqttUsername)
    opts.SetPassword(config.MqttPassword)
	opts.SetAutoReconnect(true)
    opts.SetDefaultPublishHandler(messagePubHandler)
	opts.SetKeepAlive(15)
    opts.SetWill(config.AvailableTopic, config.OfflinePayload, 0, true)
    opts.OnConnect = connectHandler
    opts.OnConnectionLost = connectLostHandler
    client = mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }
    subscribeMQTT(client)
}

func shutdownMQTT(client mqtt.Client) {
	client.Disconnect(250)
}

func updateDevice (name string, attribute string, value string) {
	log.Println("running", "aprontest", "-u", "-m", getDeviceIDFromName(name), "-t", getAttribIDFromName(attribute), "-v", value)
	cmd := exec.Command("aprontest", "-u", "-m", getDeviceIDFromName(name), "-t", getAttribIDFromName(attribute), "-v", value)
	// cmd := exec.Command("/home/sandeepc/work/personal/testing/mqttwink/testcommand.sh", "-u", "-m", getDeviceIDFromName(name), "-t", getAttribIDFromName(attribute), "-v", value)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	//cp.execFile('set_rgb', ["0", "255", "0"]);
}

func checkDatabase(init bool) error {
	log.Printf("checking database\n")
	cmd := exec.Command("sqlite3", "-csv", config.DbFile, sqlquery)
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()
    scanner := bufio.NewScanner(stdout)
    scanner.Split(bufio.ScanWords)
    for scanner.Scan() {
        output_line := scanner.Text()
		split_array := strings.Split(output_line, ",")
		if init {
			if val, ok := cached_state[split_array[0]][split_array[1]]; !ok {
				_ = val
				cached_state[split_array[0]] = map[string]string{}
				cached_state[split_array[0]][split_array[1]] = split_array[2]
				log.Printf("init - device: %s, attribute: %s, new value: %s\n", split_array[0], split_array[1], split_array[2])
			}
		} else {
			if cached_state[split_array[0]][split_array[1]] != split_array[2] {
				log.Printf("updated - device: %s, attribute: %s, new value: %s\n", split_array[0], split_array[1], split_array[2])
				publishMQTT(client, split_array[0], split_array[1], split_array[2])
				cached_state[split_array[0]][split_array[1]] = split_array[2]
			} else {
				log.Printf("not relevant : device: %s, attribute: %s, new value: %s\n", split_array[0], split_array[1], split_array[2])
				// log.Println("")
			}
		}
    }
    cmd.Wait()
	// fmt.Println(cached_state)

	return nil
}

func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) != 1 {
		log.Fatal("Usage:", os.Args, " <config file>")
	}
	log.Println("======= starting =======")
	readConfig(argsWithoutProg[0])
	initMQTT()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	if err := checkDatabase(true); err != nil {
		log.Fatal(err)
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					// log.Println("modified file:", event.Name)
					if err := checkDatabase(false); err != nil {
						log.Fatal(err)
					}				
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(config.DbFile)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}


// func database() error {
// 	db, err := sql.Open("sqlite", "/var/lib/database/apron.db")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	rows, err := db.Query("select d.masterId, s.attributeId, s.value_SET FROM zwaveDeviceState AS s,zwaveDevice AS d WHERE d.nodeId=s.nodeId AND s.attributeId IN (10,15,23)")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	for rows.Next() {
// 		var id int
// 		var aid int
// 		var name string
// 		if err = rows.Scan(&id, &aid, &name); err != nil {
// 			log.Fatal(err)
// 		}

// 		fmt.Println(id, aid, name)
// 	}

// 	if err = rows.Err(); err != nil {
// 		log.Fatal(err)
// 	}

// 	if err = db.Close(); err != nil {
// 		log.Fatal(err)
// 	}
// 	return nil
// }

package main

import (
 // "expvar"
  "flag"
  "fmt"
  "encoding/json"
  "io/ioutil"
  "net/http"
  MQTT "github.com/eclipse/paho.mqtt.golang"
  "os"
  "errors"
  "log"
)


// Structure from configuration file
type Config	struct {
        Endpoint        string          // host url
        Endpoint_port   string          // port of the Kottan server
        Channel         string          // Kottan Channel name
        Gateway         string          // Run this as HTTP server
        Gateway_port    string          // Port number to use for this HTTP
        Accountid       string          // Account ID
        Userid          string          // User Id , an account can have multiple user-ids
        Apikey          string          // userid specific apikey
        Data            string          // data file
        Log             string          // log data
}

// DefaultConfig
var DefaultConfig = Config{
	Endpoint:	"tcp://192.168.1.243", 
	Endpoint_port:	"50102", 
	Channel:	"kottan/hoot", 
	Gateway:	"false", 
	Gateway_port:	"50100",
        Accountid: 	"",
        Userid: 	"",
        Apikey: 	"",
        Data:           "",
        Log:     	"",
}


//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
  fmt.Printf("TOPIC: %s\n", msg.Topic())
  fmt.Printf("MSG: %s\n", msg.Payload())
}


func about( w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w,"Kottan Hoot Agent ...\n")
}

// Reading / Loading configuration details
func ReadConfig(configFile string) (*Config, error) {

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, errors.New("Config file does not exist.")
	} else if err != nil {
		return nil, err
	}

        var conf Config
	data, _ := ioutil.ReadFile(configFile)
	fmt.Println("Inside ReadConfig: ", configFile)
	if err := json.Unmarshal(data, &conf); err != nil {
		return nil, err
	}
	fmt.Println("End of loading\n")
	return &conf, nil
}


func main() {

    // configuration file  commandline value
    configfile := flag.String("configfile", "hoot.json", "configuration file")

   // Payload from commandline 
   payload  := flag.String("payload", "", "payload data")
   flag.Parse()

   var config Config
   file, err := os.Open(*configfile);  
   if err != nil { 
	fmt.Println(err)
   }

   decoder := json.NewDecoder(file)
   err = decoder.Decode(&config)
   if err != nil {
	fmt.Println( err )
   }

  //create a ClientOptions struct setting the broker address, clientid, turn
  //off trace output and set the default message handler
  // opts := MQTT.NewClientOptions().AddBroker("tcp://localhost:1883")
  // opts := MQTT.NewClientOptions().AddBroker("tcp://192.168.1.243:50102")
  // opts.SetClientID("kottan-pub")

  opts := MQTT.NewClientOptions().AddBroker(config.Endpoint +":"+ config.Endpoint_port)
  opts.SetClientID(config.Channel)
  opts.SetDefaultPublishHandler(f)

  //create and start a client using the above ClientOptions
  c := MQTT.NewClient(opts)
  if token := c.Connect(); token.Wait() && token.Error() != nil {
    panic(token.Error())
  }

  if payload != nil {
    // sample payload "accountid,device=1,location=rasberypi temperature=64"
    // result := c.Publish("kottan/hoot", 0, false, *payload)
    result := c.Publish(config.Channel, 0, false, *payload)
    result.Wait()
  }

  // Run as agent when agent flag is used
   if config.Gateway == "true" {
	http.HandleFunc("/", about)
	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "%#v\n", config)
	})

	http.HandleFunc("/hoot", func(w http.ResponseWriter, r *http.Request) {
		// Only post method
		if r.Method == "POST" {
			r.ParseForm()
 			payload := r.Form.Get("payload");
			fmt.Fprintf(w, payload)

			//push payload to central server
   			result := c.Publish(config.Channel, 0, false, payload)
    			result.Wait()
			
		} else {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Listening on", config.Gateway_port)
	err = http.ListenAndServe(":"+config.Gateway_port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
    }


  //unsubscribe from /go-mqtt/sample
  if token := c.Unsubscribe(config.Channel); token.Wait() && token.Error() != nil {
    fmt.Println(token.Error())
    os.Exit(1)
  }

  c.Disconnect(250)
}


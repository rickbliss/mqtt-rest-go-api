package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/julienschmidt/httprouter"
)

var i int64

var (
	addr        = flag.String("addr", ":8080", "http service address")
	data        map[string]string
	garagedoors map[string]string
)

//Pub for GDS
func getgds() {
	//Set gdmap to empty
	garagedoors = map[string]string{}
	//MQTT.DEBUG = log.New(os.Stdout, "", 0)
	//MQTT.ERROR = log.New(os.Stdout, "", 0)
	stdin := bufio.NewReader(os.Stdin)
	hostname, _ := os.Hostname() . "pub"

	server := flag.String("server", "tcp://127.0.0.1:1883", "The full URL of the MQTT server to connect to")
	topic := flag.String("topic", hostname, "Topic to publish the messages on")
	qos := flag.Int("qos", 0, "The QoS to send the messages at")
	retained := flag.Bool("retained", false, "Are the messages sent with the retained flag")
	clientid := flag.String("clientid", hostname+strconv.Itoa(time.Now().Second()), "A clientid for the connection")
	username := flag.String("username", "", "A username to authenticate to the MQTT server")
	password := flag.String("password", "", "Password to match username")
	flag.Parse()

	connOpts := MQTT.NewClientOptions().AddBroker(*server).SetClientID(*clientid).SetCleanSession(true)
	if *username != "" {
		connOpts.SetUsername(*username)
		if *password != "" {
			connOpts.SetPassword(*password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		return
	}
	fmt.Printf("Connected to %s\n", *server)

	for {
		message, err := stdin.ReadString('\n')
		if err == io.EOF {
			os.Exit(0)
		}
		client.Publish(*topic, byte(*qos), *retained, message)
	}
	fmt.Println(len(garagedoors))

	for len(garagedoors) != 2 {
		fmt.Println("waiting for 2")
		time.Sleep(1 * time.Second)
		client := MQTT.NewClient(connOpts)
	}

	//While map not > 2
	//Pub
	//update map
}

func main() {

	//MQTT.DEBUG = log.New(os.Stdout, "", 0)
	//MQTT.ERROR = log.New(os.Stdout, "", 0)
	c := make(chan os.Signal, 1)
	i = 0
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("signal received, exiting")
		os.Exit(0)
	}()
	//MQTT STUFF
	hostname, _ := os.Hostname()

	server := flag.String("server", "tcp://rickbliss.strangled.net:30845", "The full url of the MQTT server to connect to ex: tcp://127.0.0.1:1883")
	topic := flag.String("topic", "#", "Topic to subscribe to")
	qos := flag.Int("qos", 0, "The QoS to subscribe to messages at")
	clientid := flag.String("clientid", hostname+strconv.Itoa(time.Now().Second()), "A clientid for the connection")
	username := flag.String("username", "", "A username to authenticate to the MQTT server")
	password := flag.String("password", "", "Password to match username")
	flag.Parse()

	connOpts := &MQTT.ClientOptions{
		ClientID:             *clientid,
		CleanSession:         true,
		Username:             *username,
		Password:             *password,
		MaxReconnectInterval: 1 * time.Second,
		//KeepAlive:            30 * time.Second,
		TLSConfig: tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert},
	}
	connOpts.AddBroker(*server)
	connOpts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe(*topic, byte(*qos), onMessageReceived); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

	}

	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	} else {
		fmt.Printf("Connected to %s\n", *server)
	}
	flag.Parse()
	data = map[string]string{}

	r := httprouter.New()
	r.GET("/entry/:key", show)
	r.GET("/list", show)
	r.GET("/gds", gds)
	r.PUT("/entry/:key/:value", update)
	err := http.ListenAndServe(*addr, r)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}

	for {
		time.Sleep(1 * time.Second)
	}

}

func show(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	k := p.ByName("key")
	if k == "" {
		fmt.Fprintf(w, "Read list: %v", data)
		return
	}
	fmt.Fprintf(w, "Read entry: data[%s] = %s", k, data[k])
}

func gds(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	garagedoors = map[string]string{}
	go getgds()
	for len(garagedoors) != 2 {

	}
	fmt.Fprintf(w, "Read list: %v", garagedoors)
}

func update(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	k := p.ByName("key")
	v := p.ByName("value")

	data[k] = v

	fmt.Fprintf(w, "Updated: data[%s] = %s", k, data[k])
}

func onMessageReceived(client MQTT.Client, message MQTT.Message) {
	fmt.Printf("Received message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())
	if message.Topic() == "home/garage/door1" || message.Topic() == "home/garage/door2" {

		strtopic := string(message.Topic()[:])
		strpayload := string(message.Payload()[:])

		garagedoors[strtopic] = strpayload
	}

}

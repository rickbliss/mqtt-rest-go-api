package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	server      = "tcp://rickbliss.strangled.net:30845"
	keepAlive   = 5 * time.Second
	qos         = 0
	retained    = false
	clientid    = "gopublisher"
	username    = ""
	password    = ""
	connOpts    = MQTT.NewClientOptions().AddBroker(server).SetClientID(clientid).SetCleanSession(true)
	tlsConfig   = &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	client      = MQTT.NewClient(connOpts)
)

//Pub for GDS
func getgds() {
	MQTT.DEBUG = log.New(os.Stdout, "", 0)
	MQTT.ERROR = log.New(os.Stdout, "", 0)
	//Set gdmap to empty
	garagedoors = map[string]string{}

	fmt.Println(len(garagedoors))
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		//panic(token.Error())
		fmt.Println("alreadyconn")
	} else {
		fmt.Printf("Connected to %s\n", server)
	}
	//pubmqtt("home/garage/doors", "gds")
	client.Publish("home/garage/doors", byte(qos), retained, "gds")
	fmt.Println("After Publish 1")
	for len(garagedoors) != 2 {
		//pubmqtt("home/garage/doors", "gds")
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			//panic(token.Error())
			fmt.Println("alreadyconn")
			client.Publish("home/garage/doors", byte(qos), retained, "gds")
		} else {
			fmt.Printf("Connected to %s\n", server)
		}
		client.Publish("home/garage/doors", byte(qos), retained, "gds")
		fmt.Println("After Publish 2")
		time.Sleep(100 * time.Millisecond)

	}

}

func pubmqtt(topic string, message string) {

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		//panic(token.Error())
		fmt.Println("alreadyconn")
	} else {
		fmt.Printf("Connected to %s\n", server)
	}

	fmt.Println("in publish")

	client.Publish("home/garage/doors", byte(qos), retained, "gds")
	fmt.Println("after publish")
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
	topic := "#"
	connOpts.AddBroker(server)

	connOpts.KeepAlive = 10

	connOpts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe(topic, byte(qos), onMessageReceived); token.Wait() && token.Error() != nil {
			//panic(token.Error())
			fmt.Printf("already conn")
		}

	}

	//Doing this ... not sure why
	garagedoors = map[string]string{}
	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		//panic(token.Error())
		fmt.Printf("already conn")
	} else {
		fmt.Printf("Connected to %s\n", server)
	}
	flag.Parse()
	client.Publish("home/garage/doors", byte(qos), retained, "gds")
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

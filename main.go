package main

import (
	"fmt"
	"./xmpp"
	"log"
	"math/rand"
	"time"
	"flag"
)

const tagSent string = "sent"
const tagReceived string = "received"

// parameters
var msPerMsgPerUser = flag.Int("f", 1000, "milliseconds per message per user")
var totalMsgPerUser = flag.Int("t", 60, "total number of messages per user")

func main() {
	flag.Parse()
	// login all users sequencially
	xmppClients := []*xmpp.Client{}
	loginChan := make(chan string)
	for i:=1; i<=60; i++ {
		go func(id int) {
			var err error
			user := fmt.Sprintf("u_%d@jabber.hylaa.net", id)
			options := xmpp.Options{Host: "jabber.hylaa.net:5222",
				User:          user,
				Password:      "P@ssw0rd",
				NoTLS:         true,
				Debug:         false,
				Session:       false,
				Status:        "xa",
				StatusMessage: "Allen is testing",
			}

			randomSleep(5)
			xmppClient, err := options.NewClient()

			if err != nil {
				log.Fatal(err)
			} else {
				xmppClients = append(xmppClients, xmppClient)
				loginChan <- user
			}
		}(i)
	}
	// wait for all users to login
	totalLogin := 0
	for u := range loginChan{
		totalLogin += 1
		fmt.Printf("%s logs in (total: %d)\n", u, totalLogin)
		if totalLogin == 60 {
			break
		}
	}

	resultChan := make(chan string, 100000000)
	for i:=0; i<60; i++ {
		go chatbot(xmppClients[i], resultChan)
	}

	// new thread to print out result per second
	sent := 0
	received := 0
	go func() {
		counter := 0
		for {
			fmt.Printf("Time: %ds, Sent: %d, Received: %d, Rate: %f\n", counter, sent, received, float64(received)/float64(sent))
			time.Sleep(time.Second)
			counter += 1
		}
	}()

	// read result from result channel
	for r := range resultChan{
		if r == tagSent {
			sent+=1
		}else if(r == tagReceived) {
			received+=1
		}
	}
}

func randomSleep(maxSecond float64) {
	x :=time.Duration(maxSecond * float64(rand.Intn(1000))) * time.Millisecond
	time.Sleep(x)
}
func chatbot(talk *xmpp.Client, resultChan chan<- string) {
	// receive message
	go func() {
		for {
			chat, err := talk.Recv()
			if err != nil {
				log.Fatal(err)
			}
			switch chat.(type) {
			case xmpp.Chat:
				resultChan <- tagReceived
			}
		}
	}()
	// random delay, upto 2 seconds
	randomSleep(2)
	// send message
	maxInterval := float64(*msPerMsgPerUser) * 2.0 / 1000.0
	for i:=0; i<*totalMsgPerUser; i++ {
		randomUser := fmt.Sprintf("u_%d@jabber.hylaa.net", rand.Intn(60)+1)
		talk.Send(xmpp.Chat{Remote: randomUser, Type: "chat", Text: "hello"})
		resultChan <- tagSent
		randomSleep(maxInterval)
	}
}

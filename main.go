package main

import (
	"log"
	"math/rand"
	"time"
	"flag"
	"fmt"
	"./xmpp"
	"github.com/wcharczuk/go-chart"
	"github.com/chrislusf/glow/flow"
	"bytes"
	"io/ioutil"
	"regexp"
	"strconv"
)

const tagSent string = "sent"
const tagReceived string = "received"

// parameters
var msPerMsgPerUser = flag.Int("f", 100, "milliseconds per message per user")
var totalMsgPerUser = flag.Int("t", 60, "total number of messages per user")
var sampleRate = flag.Int("r", 100, "sample rate in milliseconds")
var imgFile = flag.String("o", "", "chart output of the result")
var numberOfUsers = 60 // u_1, u_2 , ... , created in advance
var totalMsg = 0

func main() {
	flag.Parse()
	totalMsg = *totalMsgPerUser * numberOfUsers

	// login all users sequencially
	xmppClients := []*xmpp.Client{}
	loginChan := make(chan string)
	for i:=1; i<=numberOfUsers; i++ {
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

			randomSleep(10) // login all users within 10 seconds
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
		if totalLogin == numberOfUsers {
			break
		}
	}

	resultChan := make(chan string, 100000000)
	latencyChan := make(chan int64, 100000000)

	for _,xmppClient := range xmppClients  {
		go chatbot(xmppClient, resultChan, latencyChan)
	}

	// new thread to print out result per second
	sent := 0
	received := 0
	xValues := []float64{}
	ySentValues := []float64{}
	yReceivedValues := []float64{}

	exitChan := make(chan string)
	go func() {
		counter := 0

		for {
			select {
			case <-exitChan:
				return;
			default:
				rate := 0.0
				if sent != 0{
					rate = float64(received)/float64(sent)
				}
				xValues = append(xValues, float64(counter))
				ySentValues = append(ySentValues, float64(sent))
				yReceivedValues = append(yReceivedValues, float64(received))
				fmt.Printf("Time: %dms, Sent: %d, Received: %d, Rate: %f\n", counter, sent, received, rate)
				time.Sleep(time.Duration(*sampleRate) * time.Millisecond)
				counter += *sampleRate
			}
		}
	}()

	// read result from result channel
	for r := range resultChan{
		if r == tagSent {
			sent+=1
		}else if(r == tagReceived) {
			received+=1
			if received == totalMsg {
				time.Sleep(5 * time.Second) // sleep a while to let the output continue
				exitChan <- "exit"
				close(exitChan)
				close(resultChan)
				close(latencyChan)
			}
		}
	}

	if *imgFile == "" {
		return
	}

	// output file name
	reg := regexp.MustCompile(`\.png$`)
	imgPrefix := reg.ReplaceAllString(*imgFile, "")
	lineChartFile := imgPrefix + ".line.png";
	pieChartFile := imgPrefix + ".pie.png";

	// line chart
	line := chart.Chart{
		XAxis: chart.XAxis{
			Name:      "time(milliseconds)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "Blue(Sent)\nGreen(received)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Name: "Sent",
				XValues:  xValues,
				YValues:  ySentValues,
			},
			chart.ContinuousSeries{
				Name: "Received",
				XValues:  xValues,
				YValues:  yReceivedValues,
			},
		},
	}

	buffer := bytes.NewBuffer([]byte{})
	fmt.Println("generating line chart ...")
	line.Render(chart.PNG, buffer)
	ioutil.WriteFile(lineChartFile, buffer.Bytes(), 0644)
	fmt.Println("Line chart is saved to " + lineChartFile)

	// parse latency datadata
	var latencyDataSrc map[string]int
	flow.New().Channel(
		latencyChan,
	).Map(func(l int64) map[string]int {
		key := strconv.FormatInt(l/1000 + 1, 10) + "s" // ceil in seconds
		return map[string]int{key : 1}
	}).Reduce(func(x map[string]int, y map[string]int) map[string]int { // merge two maps
		z := map[string]int{}
		for k,v := range x {
			z[k] = v
		}
		for k,v := range y {
			if _, ok := z[k]; ok {
				z[k] += v
			}else{
				z[k] = v
			}
		}
		return z
	}).Map(func(x map[string]int) {
		latencyDataSrc = x
	}).Run()

	// pie chart
	// convert latencyDataSrc to []chart.Value
	pieDataSrc := []chart.Value{}
	for k,v := range latencyDataSrc {
		pieDataSrc = append(pieDataSrc, chart.Value{
			Label: k,
			Value: float64(v),
		})
	}

	pie := chart.PieChart{
		Width:  512,
		Height: 512,
		Values: pieDataSrc,
	}

	buffer = bytes.NewBuffer([]byte{})
	fmt.Println("generating pie chart ...")
	pie.Render(chart.PNG, buffer)
	ioutil.WriteFile(pieChartFile, buffer.Bytes(), 0644)
	fmt.Println("Pie chart is saved to " + pieChartFile)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func randomSleep(maxSecond float64) {
	x :=time.Duration(maxSecond * float64(rand.Intn(1000))) * time.Millisecond
	time.Sleep(x)
}
func nowInUnixMilli() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond))
}
func chatbot(talk *xmpp.Client, resultChan chan<- string, latencyChan chan<- int64) {
	// receive message
	go func() {
		for {
			chat, err := talk.Recv()
			if err != nil {
				log.Fatal(err)
			}
			switch v := chat.(type) {
				case xmpp.Chat:
					resultChan <- tagReceived
					// calculate the latency
					sentTime, _ := strconv.ParseInt(v.Text, 10, 64)
					latencyMs := nowInUnixMilli() - sentTime
					latencyChan <- latencyMs
			}
		}
	}()
	// random delay, upto 2 seconds
	randomSleep(2)
	// send message
	maxInterval := float64(*msPerMsgPerUser) * 2.0 / 1000.0
	for i:=0; i<*totalMsgPerUser; i++ {
		randomUser := fmt.Sprintf("u_%d@jabber.hylaa.net", rand.Intn(numberOfUsers)+1)
		talk.Send(xmpp.Chat{Remote: randomUser, Type: "chat", Text: strconv.FormatInt(nowInUnixMilli(), 10)})
		resultChan <- tagSent
		randomSleep(maxInterval)
	}
}

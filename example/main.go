package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/centraldereservas/throttler"
)

var handler throttler.Handler

func main() {
	start := time.Now()

	// read the flags
	numRequests := flag.Int("numReq", 10, "number of requests to call in parallel")
	reqChanCap := flag.Int("reqChanCap", 10, "capacity of the requests channel")
	maxCallsPerSecond := flag.Int("maxCallsPerSec", 2, "maximal number of calls per second")
	guardTimeInMs := flag.Int("guardTimeInMs", 50, "extra time to wait between two consecutive calls (in miliseconds)")
	reqTimeoutInMs := flag.Int("reqTimeoutInMs", 10000, "request timeout (in miliseconds)")
	globalTimeoutInMs := flag.Int("globalTimeoutInMs", 30000, "global timeout (in miliseconds) for sending all the requests")
	verbose := flag.Bool("verbose", true, "if true prints information about the requests fulfilled by the throttler handler (name, timestamp, order)")

	flag.Parse()
	guardTime := time.Duration(*guardTimeInMs) * time.Millisecond
	reqTimeout := time.Duration(*reqTimeoutInMs) * time.Millisecond
	globalTimeout := time.Duration(*globalTimeoutInMs) * time.Millisecond

	fmt.Println("Throttler started")

	// initialize the handler
	handler = initHandler(*maxCallsPerSecond, guardTime, *reqChanCap, *verbose, globalTimeout)
	req := createRequest()
	ctx := context.Background()
	c := make(chan *http.Response)

	// Generate some requests and queue them to the requests channel to be processed
	// when it corresponds (according to the maxCallsPerSecond configuration)
	for i := 0; i < *numRequests; i++ {
		name := "Task " + strconv.Itoa(i)
		go func() {
			c <- handleRequest(ctx, name, req, reqTimeout)
		}()
	}
	fmt.Printf("%d request(s) pending to be processed at Rate = (1 call / %v).\n\n", *numRequests, handler.Rate())

	// Wait for receiving the responses from the channel and process each of them.
	// If a time out occurs, break the for loop.
	timeout := time.After(globalTimeout)
	for i := 0; i < *numRequests; i++ {
		select {
		case result := <-c:
			processResponse(i, result)
		case <-timeout:
			fmt.Printf("timed out")
			return
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("\nElapsed time: %v\n", elapsed)
}

// initHandler creates a new instance of a throttler.Handler
func initHandler(maxCallsPerSecond int, guardTime time.Duration, requestChannelCapacity int, verbose bool, globalTimeoutInMs time.Duration) throttler.Handler {
	rate, err := throttler.NewRateByCallsPerSecond(maxCallsPerSecond, guardTime)
	if err != nil {
		log.Fatalf("unable to create a rate: %v", err)
	}
	handler, err := throttler.NewHandler(rate, requestChannelCapacity, verbose)
	if err != nil || handler == nil {
		log.Fatalf("unable to create a new handler: %v", err)
	}
	handler.SetClient(&http.Client{Timeout: globalTimeoutInMs})
	handler.StartRequestsHandler()
	return handler
}

// createRequest creates a request to a test API to retrieve the current datetime
func createRequest() *http.Request {
	url := "http://date.jsontest.com/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("unable to create request: %v", err)
	}
	return req
}

// handleRequest queues the http.Request to the handler requests channel
func handleRequest(ctx context.Context, name string, req *http.Request, requestTimeout time.Duration) *http.Response {
	res, err := handler.Queue(ctx, name, req, requestTimeout)
	if err != nil {
		log.Fatalf("unable to queue the request: %v", err)
	}
	return res
}

// processResponse extracts the body from the http.Response and print it to the standard output
func processResponse(index int, res *http.Response) {
	if res.StatusCode != http.StatusOK {
		log.Fatalf("http status code is not OK: %s", res.Status)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Error reading response: %s", err)
	}
	if len(body) == 0 {
		log.Fatalf("response with empty body. Status: %s", res.Status)
	}
}

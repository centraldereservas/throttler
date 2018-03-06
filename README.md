# throttler

Provides a throttle request channel for Go that controls request rate limit in order to prevent exceeding a predefined API quota.


## Installation

```sh
go get -u bitbucket.org/cdr-danielgil/throttler
```

## Motivation
 
Why we need to control the request rate? 
Because some API's limits the maximal number of calls that a client can request in order to control 
the received traffic. If this rate limit is overtaken then the sender will receive an HTTP Error from the server, 
for example a `403 Developer Over Rate`.

To avoid this situation the throttler package uses a mechanism to regulate the request flow to the server based on the leaky bucket 
algorithm where the bucket is represented by a buffered channel.


## Documentation

The throttler package has 3 main components: `Rate`, `Handler` and `Queue`.

API documentation is available on [godoc.org][doc]. 


### Rate
The `Rate` defines the limits to communicate with a server that can not be broken. The rate duration is calculated as 

```Rate Duration = period + guardTime```

where `period` represents the frequency that we should send the requests and it is calculated by any 
of the initializers `NewRateByCallsPerSec`, `NewRateByCallsPerMinute` or `NewRateByCallsPerHour` 
and `guardTime` is an extra time to wait beetween two consecutive calls.

### Handler
The `Handler` is the responsible to start in a new goroutine a mechanism called `requestsHandler` which controls
that the requests are fulfilled at the proper time respecting the `Rate` limits.

### Queue
The `Queue` function queues a new `throttler.Request` (which contains an `http.Request`) to the shared 
requests channel and blocks the thread until the `requestsHandler` decides that the request can be processed. 
When this happens, the function `fulfillRequest` is called which internally calls the `http.Client.Do(http.Request)`. 
Finally the `Queue` function returns an `http.Response`.


## Usage
To use this package the first thing we have to do is create a `Rate` using any of the available constructors:
```go
rate, err := throttler.NewRateByCallsPerSec(maxCallsPerSecond, guardTime)
rate, err := throttler.NewRateByCallsPerMinute(maxCallsPerMin, guardTime)
rate, err := throttler.NewRateByCallsPerHour(maxCallsPerHour, guardTime)
```
where `maxCallsPerSecond`, `maxCallsPerMin` and `maxCallsPerHour` are integers that define the maximal number of 
requests allowed to send to the client, and `guardTime` is an extra time to wait beetween two consecutive calls.

Then we create an instance of a throttler handler passing the rate:

```go
handler, err := throttler.NewHandler(rate, requestChannelCapacity, verbose)
```
where `requestChannelCapacity` is the capacity of the channel that will contains all the queued requests and 
`verbose` is a boolean that displays debug information in the standard output if`true`. 

Once we have an instance of the handler we can start the requests handler which will be listening for new requests
from the requests channel:
```go
handler.StartRequestsHandler()
```

After this we can queue new requests to the channel:

```go
res, err := handler.Queue(ctx, name, req, timeout)
```

where `ctx` is the context (used for cancellation propagation), `name` is an optional field used just for verbose, 
`req` of type `http.Request` is the request and `timeout` is the request timeout `time.Duration`.

### Concurrent Requests
In some situations we need to send multiple calls in parallel and we would like to avoid blocking the thread, for this case
we can achieve this by using goroutines. If we encapsulate the `Queue` call into a function like this:

```go
func handleRequest(ctx context.Context, name string, req *http.Request, timeout time.Duration) *http.Response {
	res, err := handler.Queue(ctx, name, req, timeout)
	if err != nil {
		log.Fatalf("unable to queue the request: %v", err)
	}
	return res
}
```

then we could call the `handleRequest` in a new goroutine and send as many requests as we need.
Later we can wait for the responses using a select case statement which controls if a global timeout has expired 
to stop waiting:

```go
c := make(chan *http.Response)
for i := 0; i < numRequests; i++ {
	name := "Task " + strconv.Itoa(i)
	go func() {
		c <- handleRequest(ctx, name, req, reqTimeout)
	}()
}
timeout := time.After(globalTimeout)
for i := 0; i < numRequests; i++ {
	select {
	case result := <-c:
		processResponse(i, result)
	case <-timeout:
		fmt.Printf("timed out")
		return
	}
}
```

### Set http.Client
By default the package sets the internal `http.Client` to the `DefaultClient` but sometimes it is desired to customize the client
specifying timeouts, redirect policy, proxies or simply to be used with Google App Engine. We can set the `http.Client` by calling
the function `SetClient(client *http.Client)` just before the `StartRequestsHandler()`.

Example:
```go
handler, err := throttler.NewHandler(rate, requestChannelCapacity, verbose)
client := &http.Client{
	Timeout:       10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	Transport:     &http.Transport{TLSHandshakeTimeout: 5 * time.Second},
}
handler.SetClient(client)
handler.StartRequestsHandler()
```

## Example

There is a complete example under the `/example/` folder in the source code that shows how the package is used. 
The `/example/main.go` file accepts the following flags:

- **numReq**: Number of requests to call in parallel (default: 10).
- **reqChanCap**: Capacity of the requests channel (default: 10).
- **maxCallsPerSec**: Maximal number of calls per second (default: 2).
- **guardTimeInMs**: Extra time in miliseconds to wait between two consecutive calls (default: 50).
- **reqTimeoutInMs**: Request timeout in miliseconds (default: 10000, it is 10 seconds).
- **globalTimeoutInMs**: Global timeout in miliseconds for sending all the requests (default: 30000, it is 30 seconds). 
- **verbose**: If true prints information about the requests fulfilled by the throttler handler: name, timestamp, order (default: true). 


Output:

```sh
$ go run ./example/main.go
Throttler started
10 request(s) pending to be processed at Rate = (1 call / 550ms).

[2018-02-14 08:34:16.540434 +0100 CET m=+0.551561077] got ticket; Fulfilling Request [Task 0]
[2018-02-14 08:34:16.541076 +0100 CET m=+0.552204278] Request fulfilled [Task 0]
[2018-02-14 08:34:17.092734 +0100 CET m=+1.103889840] got ticket; Fulfilling Request [Task 1]
[2018-02-14 08:34:17.092874 +0100 CET m=+1.104030095] Request fulfilled [Task 1]
[2018-02-14 08:34:17.642794 +0100 CET m=+1.653978019] got ticket; Fulfilling Request [Task 9]
[2018-02-14 08:34:17.642918 +0100 CET m=+1.654101990] Request fulfilled [Task 9]
[2018-02-14 08:34:18.193137 +0100 CET m=+2.204348959] got ticket; Fulfilling Request [Task 2]
[2018-02-14 08:34:18.193214 +0100 CET m=+2.204425654] Request fulfilled [Task 2]
[2018-02-14 08:34:18.739966 +0100 CET m=+2.751206224] got ticket; Fulfilling Request [Task 5]
[2018-02-14 08:34:18.740035 +0100 CET m=+2.751275318] Request fulfilled [Task 5]
[2018-02-14 08:34:19.29534 +0100 CET m=+3.306607420] got ticket; Fulfilling Request [Task 3]
[2018-02-14 08:34:19.295405 +0100 CET m=+3.306673507] Request fulfilled [Task 3]
[2018-02-14 08:34:19.842653 +0100 CET m=+3.853949471] got ticket; Fulfilling Request [Task 6]
[2018-02-14 08:34:19.842755 +0100 CET m=+3.854050567] Request fulfilled [Task 6]
[2018-02-14 08:34:20.395302 +0100 CET m=+4.406625795] got ticket; Fulfilling Request [Task 4]
[2018-02-14 08:34:20.395364 +0100 CET m=+4.406688232] Request fulfilled [Task 4]
[2018-02-14 08:34:20.943651 +0100 CET m=+4.955003116] got ticket; Fulfilling Request [Task 7]
[2018-02-14 08:34:20.94371 +0100 CET m=+4.955062235] Request fulfilled [Task 7]
[2018-02-14 08:34:21.492474 +0100 CET m=+5.503854301] got ticket; Fulfilling Request [Task 8]
[2018-02-14 08:34:21.492548 +0100 CET m=+5.503927745] Request fulfilled [Task 8]

Elapsed time: 5.820121229s
```

## Tests

The file `handler_test.go` contains some test cases for testing `NewHandler` and `Enqueue` functions.

Use the following command to run all the tests:

```sh
$ go test -v
```

Output:
```sh
$ go test -v
=== RUN   TestNewHandler
=== RUN   TestNewHandler/Positive_TC
=== RUN   TestNewHandler/Negative_TC:_maxCallsPerSecond_zero
=== RUN   TestNewHandler/Negative_TC:_maxCallsPerSecond
=== RUN   TestNewHandler/Negative_TC:_guardTime
=== RUN   TestNewHandler/Positive_TC:_guardTime_zero
=== RUN   TestNewHandler/Negative_TC:_reqChanCapacity
--- PASS: TestNewHandler (0.00s)
    --- PASS: TestNewHandler/Positive_TC (0.00s)
    --- PASS: TestNewHandler/Negative_TC:_maxCallsPerSecond_zero (0.00s)
    --- PASS: TestNewHandler/Negative_TC:_maxCallsPerSecond (0.00s)
    --- PASS: TestNewHandler/Negative_TC:_guardTime (0.00s)
    --- PASS: TestNewHandler/Positive_TC:_guardTime_zero (0.00s)
    --- PASS: TestNewHandler/Negative_TC:_reqChanCapacity (0.00s)
=== RUN   TestEnqueue
=== RUN   TestEnqueue/Positive_TC
--- PASS: TestEnqueue (1.53s)
    --- PASS: TestEnqueue/Positive_TC (1.53s)
PASS
ok      bitbucket.org/differenttravel/pase-common/throttler     1.547s
```


## Limitations
- Not working when using multiple application instances running in different servers. Solution: a distributed rate limit control 
  using a master/slave throttle where the master is the only one responsible for the leaky bucket time control.
- The fulfillRequest function always calls `http.Client.Do(http.Request)` but the package could accept the function that has to be called 
  in the `throttler.Request`. The problem is that the response channel (stored in the `throttler.Request`) will be unknown in compile time and
  it could be different from call to call.


## References
* [Rate Limiting](https://github.com/golang/go/wiki/RateLimiting).
* [Rate Limiting Service Calls in Go](https://medium.com/@KevinHoffman/rate-limiting-service-calls-in-go-3771c6b7c146) by Kevin Hoffman.



## License

This project is under the [MIT License][mit].
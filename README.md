# throttler  [![Build Status](https://travis-ci.org/centraldereservas/throttler.svg?branch=master)](https://travis-ci.org/centraldereservas/throttler) [![Coverage Status](https://coveralls.io/repos/github/centraldereservas/throttler/badge.svg?branch=master)](https://coveralls.io/github/centraldereservas/throttler?branch=master) [![Report card](https://goreportcard.com/badge/github.com/centraldereservas/throttler)](https://goreportcard.com/report/github.com/centraldereservas/throttler) ![Project status](https://img.shields.io/badge/version-0.0.3-green.svg)  ![Project dependencies](https://img.shields.io/badge/dependencies-none-green.svg) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![GoDoc](https://godoc.org/github.com/centraldereservas/throttler?status.svg)](https://godoc.org/github.com/centraldereservas/throttler)

Provides a throttle request channel for Go that controls request rate limit in order to prevent exceeding a predefined API quota.

## Installation

```sh
go get github.com/centraldereservas/throttler
```

## Motivation

Why we need to control the request rate?

Because some APIs limit the maximal number of calls that a client can request in order to control the received traffic. If this rate limit is overtaken then the sender will receive an HTTP Error from the server, for example a `403 Developer Over Rate`.

To avoid this situation the `throttler` package uses a mechanism to regulate the request flow to the server based on the leaky bucket algorithm where the bucket is represented by a buffered channel.

## Documentation

The throttler working flow has the following steps:

* Initialization: `Rate` creation passed to the throttler constructor `New`.
* Start the service with `Run`.
* Add new requests to be processed with `Queue`.

API documentation is available on [godoc.org][doc].

### Rate

`Rate` returns the minimal rate duration to communicate with a server that can not be overpassed. It is calculated as

```Rate Duration = period + guardTime```

where `period` represents the frequency that we should send the requests and `guardTime` is an extra time to wait beetween two consecutive calls.

The available `Rate` constructors are `NewRateByCallsPerSecond`, `NewRateByCallsPerMinute` or `NewRateByCallsPerHour`.

### New

The throttler constructor `New` is the responsible for initializing the requests channel and configuring the listener for this channel based on the `Rate` passed.

### Run

It starts a mechanism called `listener` in a new goroutine which controls that the requests received from the requests channel are fulfilled at the proper time respecting the `Rate` limits.

### Queue

The `Queue` function queues a new `throttler.Request` (which contains an `http.Request`) to the shared requests channel and blocks the thread until the `listener` decides that the request can be processed. When this happens, the function `fulfill` is called which internally calls the `http.Client.Do(http.Request)`. Finally the `Queue` function returns an `http.Response`.


## Usage

To use this package the first thing we have to do is create a `Rate` instance using any of the available constructors:

```go

rate, err := throttler.NewRateByCallsPerSecond(maxCallsPerSecond, guardTime)
rate, err := throttler.NewRateByCallsPerMinute(maxCallsPerMin, guardTime)
rate, err := throttler.NewRateByCallsPerHour(maxCallsPerHour, guardTime)

```

where `maxCallsPerSecond`, `maxCallsPerMin` and `maxCallsPerHour` are integers that define the maximal number of requests allowed to send to the client, and `guardTime` is an extra time to wait beetween two consecutive calls.

Then we create an instance of a throttler passing the rate:

```go

t, err := throttler.New(rate, requestChannelCapacity, client, verbose)

```

where `requestChannelCapacity` is the capacity of the channel that will contains all the queued requests, `client` is an optional parameter of type `*http.Client` and  `verbose` is a boolean that displays debug information in the standard output if `true`.

If the client param is nil is set to `http.DefaultClient`. The throttler allows to customize the `http.Client` because sometimes it is desired to specify timeouts, configure a redirect policy, proxies or simply to be used with Google App Engine.

Once we have an instance of the throttler we can activate the listener which will be waiting for new requests from the requests channel:
```go

t.Run()

```

After this we are ready to start queuing new requests to the channel:

```go

res, err := t.Queue(ctx, name, req, timeout)

```

where `ctx` is the context (used for cancellation propagation), `name` is an optional field used just for verbose, `req` is the request of type `http.Request` and `timeout` is the request timeout of type `time.Duration`.

### Concurrent Requests
In some situations we need to send multiple calls in parallel and we would like to avoid blocking the thread, for this case
we can achieve this by using goroutines. If we encapsulate the `Queue` call into a function like this:

```go

func handleRequest(ctx context.Context, name string, req *http.Request, timeout time.Duration) *http.Response {
    cres, err := t.Queue(ctx, name, req, timeout)
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


## Example

There is a complete example under the `/example/` folder in the source code that shows how the package is used. The `/example/main.go` file accepts the following flags:

* **numReq**: Number of requests to call in parallel (default: 10).
* **reqChanCap**: Capacity of the requests channel (default: 10).
* **maxCallsPerSec**: Maximal number of calls per second (default: 2).
* **guardTimeInMs**: Extra time in miliseconds to wait between two consecutive calls (default: 50).
* **reqTimeoutInMs**: Request timeout in miliseconds (default: 10000, it is 10 seconds).
* **globalTimeoutInMs**: Global timeout in miliseconds for sending all the requests (default: 30000, it is 30 seconds).
* **verbose**: If true prints information about the requests fulfilled by the throttler: name, timestamp, order (default: true).


Output:

```sh

$ go run ./example/main.go
Throttler started
10 request(s) pending to be processed at Rate = (1 call / 550ms).

[2018-03-16 11:38:20.797001121 +0100 CET m=+0.554512486] got ticket; Fulfilling Request [Task 0]
[2018-03-16 11:38:20.797321571 +0100 CET m=+0.554832931] Request fulfilled [Task 0]
[2018-03-16 11:38:21.348416765 +0100 CET m=+1.105919716] got ticket; Fulfilling Request [Task 2]
[2018-03-16 11:38:21.348537267 +0100 CET m=+1.106040216] Request fulfilled [Task 2]
[2018-03-16 11:38:21.898448991 +0100 CET m=+1.655943551] got ticket; Fulfilling Request [Task 1]
[2018-03-16 11:38:21.898548044 +0100 CET m=+1.656042602] Request fulfilled [Task 1]
[2018-03-16 11:38:22.443900051 +0100 CET m=+2.201386288] got ticket; Fulfilling Request [Task 5]
[2018-03-16 11:38:22.443991627 +0100 CET m=+2.201477863] Request fulfilled [Task 5]
[2018-03-16 11:38:22.996845058 +0100 CET m=+2.754322858] got ticket; Fulfilling Request [Task 3]
[2018-03-16 11:38:22.996902877 +0100 CET m=+2.754380676] Request fulfilled [Task 3]
[2018-03-16 11:38:23.547879646 +0100 CET m=+3.305349039] got ticket; Fulfilling Request [Task 7]
[2018-03-16 11:38:23.547948586 +0100 CET m=+3.305417978] Request fulfilled [Task 7]
[2018-03-16 11:38:24.093824873 +0100 CET m=+3.851285936] got ticket; Fulfilling Request [Task 6]
[2018-03-16 11:38:24.093901681 +0100 CET m=+3.851362743] Request fulfilled [Task 6]
[2018-03-16 11:38:24.645247933 +0100 CET m=+4.402700583] got ticket; Fulfilling Request [Task 8]
[2018-03-16 11:38:24.645312707 +0100 CET m=+4.402765356] Request fulfilled [Task 8]
[2018-03-16 11:38:25.195285184 +0100 CET m=+4.952729442] got ticket; Fulfilling Request [Task 9]
[2018-03-16 11:38:25.195345115 +0100 CET m=+4.952789372] Request fulfilled [Task 9]
[2018-03-16 11:38:25.747991251 +0100 CET m=+5.505427075] got ticket; Fulfilling Request [Task 4]
[2018-03-16 11:38:25.748063331 +0100 CET m=+5.505499154] Request fulfilled [Task 4]

Elapsed time: 5.821642877s

```

## Tests

### client_test.go

Contains a test case for testing the `send` function.


### export_test.go

Contains some alias to be able to access privave functions just for testing.

### fulfiller_test.go

Contains a test case for testing the `fulfill` function.

### listener_test.go

Contains a test case for testing the `listen` function.


### handler_test

The file `handler_test.go` contains some test cases for testing the functions `NewHandler`, `SetClient`, `Run` and `Queue`.

### rate_test.go

Contains test cases for testing the rate functions `NewRateByCallsPerSecond`, `NewRateByCallsPerMinute`, `NewRateByCallsPerHour` and `CalculateRate`.

### throttler_test.go

Contains test cases for testing the functions `New`, `Rate`, `Run` and `Queue`.


### Run all tests

Use the following command to run all the tests:

```sh

go test -v -race

```

Output:
```sh

$ go test -race
[2018-03-16 11:37:45.443431875 +0100 CET m=+1.575356426] got ticket; Fulfilling Request [Positive TC]
[2018-03-16 11:37:45.444484452 +0100 CET m=+1.576408987] Request fulfilled [Positive TC]
PASS
[2018-03-16 11:37:46.785455926 +0100 CET m=+2.917360001] got ticket; Fulfilling Request [Negative TC: force timeout in Queue]
[2018-03-16 11:37:46.785917845 +0100 CET m=+2.917821913] Request fulfilled [Negative TC: force timeout in Queue]
ok      github.com/centraldereservas/throttler  3.389s

```

## Limitations

* If we have multiple application instances could be possible to overtake the rate limit because the instances do not share the requests channel.
* The fulfill function always calls `http.Client.Do(http.Request)`.


## References

* [Rate Limiting](https://github.com/golang/go/wiki/RateLimiting)
* [Rate Limiting Service Calls in Go](https://medium.com/@KevinHoffman/rate-limiting-service-calls-in-go-3771c6b7c146) by Kevin Hoffman
* [Mocking in Go (dotGo 2014)](https://youtu.be/2_FMbcQJg0c) by Gabriel Aszalos


## License

This project is under the [MIT License][mit].

[mit]: https://github.com/centraldereservas/throttler/blob/master/LICENSE
[doc]: https://godoc.org/github.com/centraldereservas/throttler
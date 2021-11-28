# Event is a concurrent Event and Error Stream Implementation for Go

[![Build & Test Action Status](https://github.com/devnw/event/actions/workflows/build.yml/badge.svg)](https://github.com/devnw/event/actions)
[![Go Report Card](https://goreportcard.com/badge/go.devnw.com/event)](https://goreportcard.com/report/go.devnw.com/event)
[![codecov](https://codecov.io/gh/devnw/event/branch/main/graph/badge.svg)](https://codecov.io/gh/devnw/event)
[![Go Reference](https://pkg.go.dev/badge/go.devnw.com/event.svg)](https://pkg.go.dev/go.devnw.com/event)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

The `event` package provides a concurrent Event and Error Stream Implementation
for Go. It is designed using concurrency patterns and best practices to ensure
that it properly functions in highly parallel environments as well as
sequential.

The implementation provides a `Publisher` which is used to publish events as
well as ReadEvents and ReadErrors to allow subscribers to access read-only
channels of those events and errors.

**NOTE:** Publisher will *only* publish events when ReadEvents or ReadErrors are
called to ensure that there are no wasted cycles waiting for subscribers.
To that end the methods for publishing said errors/Events will return an error
if the ReadEvents/Errors are not called.

## Use

```bash
go get -u go.devnw.com/event
```

### Create a Publisher  

```go
import "go.devnw.com/event"
...

publisher := NewPublisher(ctx)
defer func() {
    err := publisher.Close()
    if err != nil {
        t.Errorf("Publisher.Close() failed: %v", err)
    }
}()
```

### Create A Handler for both the Error and Event Streams

```go

func main() {
    ...
    publisher := NewPublisher(ctx)
    ...

    // Create a subscriber to read events and errors from the publisher
    // See `Example **Subscriber**` below for more details on possible
    // subscriber implementations.
    Subscribe(
        ctx,

        // The caller supplies the buffer to use when reading events/errors
        // this buffer is only applied on the first call as that is
        // what creates the underlying channel in the implementation.
        publisher.ReadEvents(buffer).Interface(),
        publisher.ReadErrors(buffer).Interface(),
    )

    ...
}
```

### Example Subscriber

```go
// This is an example of a subscriber that will handle both events and errors
// from the incoming channels. 
func Subscribe(ctx context.Context, streams ...<-chan interface{}) {
    for _, stream := range streams {
        go func(stream <-chan interface{}) {
            for {
                case <-ctx.Done():
                    return
                case e, ok := <-stream:
                    if !ok {
                        return
                    }

                    switch event := e.(type) {
                        case error:
                            // Do something with the error
                        case Event:
                            // Do something with the event
                    }
            }
        }(stream)
    }
}
```

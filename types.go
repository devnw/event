package event

type ErrorStream <-chan error

func (e ErrorStream) Interface() <-chan interface{} {
	out := make(chan interface{})

	go func() {
		for err := range e {
			out <- err
		}
	}()

	return out
}

type ErrorWriter chan<- error

// nolint:revive
type EventStream <-chan Event

func (e EventStream) Interface() <-chan interface{} {
	out := make(chan interface{})

	go func() {
		for event := range e {
			out <- event
		}
	}()

	return out
}

// nolint:revive
type EventWriter chan<- Event

// nolint:revive
type EventFunc func() Event

type ErrorFunc func() error

type Combo interface {
	error
	Event
}

type Event interface {
	Event() string
}

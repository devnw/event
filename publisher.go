package event

import (
	"context"
	"errors"
	"sync"
)

// NewPublisher creates a new publisher for serving Event and Error streams
func NewPublisher(ctx context.Context) *Publisher {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(ctx)

	return &Publisher{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Publisher provides the ability to publish events and errors in a thread-safe
// concurrent manner using best practices.
type Publisher struct {
	ctx    context.Context
	cancel context.CancelFunc
	pubWg  sync.WaitGroup

	eventsMu sync.RWMutex
	events   chan Event

	errorsMu sync.RWMutex
	errors   chan error
}

// ReadEvents returns a stream of published events
func (p *Publisher) ReadEvents(buffer int) EventStream {
	if buffer < 0 {
		buffer = 0
	}

	p.eventsMu.Lock()
	defer p.eventsMu.Unlock()

	if p.events == nil {
		p.events = make(chan Event, buffer)
	}

	return p.events
}

// ReadErrors returns a stream of published errors
func (p *Publisher) ReadErrors(buffer int) ErrorStream {
	if buffer < 0 {
		buffer = 0
	}

	p.errorsMu.Lock()
	defer p.errorsMu.Unlock()

	if p.errors == nil {
		p.errors = make(chan error, buffer)
	}

	return p.errors
}

// EventFunc Accepts an EventFunc type as a parameter and executes it only
// if there are subscribers to the underlying event channel allowing for delayed
// data rendering of an event.
func (p *Publisher) EventFunc(ctx context.Context, fn EventFunc) (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	p.eventsMu.Lock()
	defer p.eventsMu.Unlock()

	if p.events == nil {
		return err
	}

	select {
	case <-p.ctx.Done():
		err = p.ctx.Err()
	case <-ctx.Done():
		err = ctx.Err()
	case p.events <- fn():
	}

	return err
}

// ErrorFunc Accepts an ErrorFunc type as a parameter and executes it only
// if there are subscribers to the underlying error channel allowing for delayed
// data rendering of an error.
func (p *Publisher) ErrorFunc(ctx context.Context, fn ErrorFunc) (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	p.errorsMu.Lock()
	defer p.errorsMu.Unlock()

	if p.errors == nil {
		return err
	}

	select {
	case <-p.ctx.Done():
		err = p.ctx.Err()
	case <-ctx.Done():
		err = ctx.Err()
	case p.errors <- fn():
	}

	return err
}

// Errors accepts a number of error streams and forwards them to the
// publisher. (Fan-In)
func (p *Publisher) Errors(ctx context.Context, errs ...ErrorStream) error {
	p.errorsMu.Lock()
	defer p.errorsMu.Unlock()

	if p.errors == nil {
		return errors.New("no listener for errors")
	}

	for _, err := range errs {
		if err == nil {
			continue
		}

		p.pubWg.Add(1)

		go func(err ErrorStream) {
			defer p.pubWg.Done()
			for {
				select {
				case <-p.ctx.Done():
					return
				case <-ctx.Done():
					return
				case e, ok := <-err:
					if !ok {
						return
					}

					select {
					case <-p.ctx.Done():
						return
					case <-ctx.Done():
						return
					case p.errors <- e:
					}
				}
			}
		}(err)
	}

	return nil
}

// Events accepts a number of event streams and forwards them to the
// event writer.(Fan-In)
func (p *Publisher) Events(ctx context.Context, events ...EventStream) error {
	p.eventsMu.RLock()
	defer p.eventsMu.RUnlock()

	if p.events == nil {
		return errors.New("no listener for events")
	}

	for _, event := range events {
		if event == nil {
			continue
		}

		p.pubWg.Add(1)

		go func(event EventStream) {
			defer p.pubWg.Done()

			for {
				select {
				case <-p.ctx.Done():
					return
				case <-ctx.Done():
					return
				case e, ok := <-event:
					if !ok {
						return
					}

					select {
					case <-ctx.Done():
						return
					case p.events <- e:
					}
				}
			}
		}(event)
	}

	return nil
}

// Split accepts a channel of interface types and splits them into event and
// error streams. (Fan-Out)
// nolint: gocyclo
func (p *Publisher) Split(ctx context.Context, in <-chan interface{}) error {
	if in == nil {
		return errors.New("incoming data channel is nil")
	}

	p.eventsMu.Lock()
	defer p.eventsMu.Unlock()

	if p.events == nil {
		return errors.New("no listener for events")
	}

	p.errorsMu.Lock()
	defer p.errorsMu.Unlock()

	if p.errors == nil {
		return errors.New("no listener for errors")
	}

	p.pubWg.Add(1)

	go func() {
		defer p.pubWg.Done()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ctx.Done():
				return
			case data, ok := <-in:
				if !ok {
					return
				}

				switch e := data.(type) {
				case nil:
					continue
				// TODO: Should this concatenate all the event and error?
				case Combo, error:
					// Force cast to error
					err := e.(error)

					select {
					case <-p.ctx.Done():
						return
					case <-ctx.Done():
						return
					case p.errors <- err:
					}
				case Event:
					select {
					case <-p.ctx.Done():
						return
					case <-ctx.Done():
						return
					case p.events <- e:
					}
				default:
					// NOTE: SKIP UNKNOWN TYPE
				}
			}
		}
	}()

	return nil
}

// Close closes the event and error streams
func (p *Publisher) Close() (err error) {
	defer func() {
		err = recoverErr(err, recover())
	}()

	p.cancel()
	<-p.ctx.Done()

	p.eventsMu.Lock()
	defer p.eventsMu.Unlock()

	p.errorsMu.Lock()
	defer p.errorsMu.Unlock()

	// Only attempt to close if it's non-nil
	if p.events != nil {
		defer close(p.events)
	}

	// Only attempt to close if it's non-nil
	if p.errors != nil {
		defer close(p.errors)
	}

	p.pubWg.Wait()

	return err
}

package event

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"
)

type testEvent struct{ msg string }

func (e *testEvent) Event() string {
	return e.msg
}

type testCombo struct{ msg string }

func (c *testCombo) Error() string {
	return c.msg
}

func (c *testCombo) Event() string {
	return c.msg
}

// nolint:staticcheck
func Test_NewPublisher(t *testing.T) {
	publisher := NewPublisher(context.Background())
	if publisher == nil {
		t.Errorf("NewPublisher() failed")
	}

	if publisher.ctx == nil || publisher.cancel == nil {
		t.Errorf("NewPublisher() failed")
	}
}

// nolint:staticcheck
func Test_NewPublisher_NilCtx(t *testing.T) {
	publisher := NewPublisher(nil)
	if publisher == nil {
		t.Errorf("NewPublisher() failed")
	}

	if publisher.ctx == nil || publisher.cancel == nil {
		t.Errorf("NewPublisher() failed")
	}
}

func Test_Publisher_Read(t *testing.T) {
	testdata := map[string]struct {
		msgs   []interface{}
		errs   map[string]bool
		events map[string]bool
	}{
		"single event": {
			msgs: []interface{}{&testEvent{msg: "test"}},
			events: map[string]bool{
				"test": true,
			},
		},
		"single error": {
			msgs: []interface{}{errors.New("test")},
			errs: map[string]bool{
				"test": true,
			},
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			publisher := NewPublisher(ctx)
			defer func() {
				err := publisher.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			events := publisher.ReadEvents(len(test.events))
			errs := publisher.ReadErrors(len(test.errs))

			in := make(chan interface{})
			defer close(in)

			publisher.Split(ctx, in)
			for _, msg := range test.msgs {
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case in <- msg:
				}
			}

			for i := 0; i < len(test.errs)+len(test.events); i++ {
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case event, ok := <-events:
					if !ok {
						t.Fatal("events channel closed")
					}

					if _, exists := test.events[event.Event()]; !exists {
						t.Errorf("expected event %s; doesn't exist", event.Event())
					}
				case err, ok := <-errs:
					if !ok {
						t.Fatal("errors channel closed")
					}

					if _, exists := test.errs[err.Error()]; !exists {
						t.Errorf("expected error %s; doesn't exist", err.Error())
					}
				}
			}
		})
	}
}

func Test_Publisher_Read_Interfaces(t *testing.T) {
	testdata := map[string]struct {
		msgs   []interface{}
		errs   map[string]bool
		events map[string]bool
	}{
		"single event": {
			msgs: []interface{}{&testEvent{msg: "test"}},
			events: map[string]bool{
				"test": true,
			},
		},
		"single error": {
			msgs: []interface{}{errors.New("test")},
			errs: map[string]bool{
				"test": true,
			},
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			publisher := NewPublisher(ctx)
			defer func() {
				err := publisher.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			events := publisher.ReadEvents(len(test.events)).Interface()
			errs := publisher.ReadErrors(len(test.errs)).Interface()

			in := make(chan interface{})
			defer close(in)

			publisher.Split(ctx, in)
			for _, msg := range test.msgs {
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case in <- msg:
				}
			}

			for i := 0; i < len(test.errs)+len(test.events); i++ {
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case data, ok := <-events:
					if !ok {
						t.Fatal("events channel closed")
					}

					event, ok := data.(Event)
					if !ok {
						t.Fatalf("event is not an Event | %T", data)
					}

					if _, exists := test.events[event.Event()]; !exists {
						t.Errorf("expected event %s; doesn't exist", event.Event())
					}
				case data, ok := <-errs:
					if !ok {
						t.Fatal("errors channel closed")
					}

					err, ok := data.(error)
					if !ok {
						t.Fatalf("err is not an error | %T", data)
					}

					if _, exists := test.errs[err.Error()]; !exists {
						t.Errorf("expected error %s; doesn't exist", err.Error())
					}
				}
			}
		})
	}
}

type wrappedError interface {
	error
	Unwrap() error
}

func Test_recoverErr(t *testing.T) {
	testdata := map[string]struct {
		value      interface{}
		underlying error
		expected   error
	}{
		"nil": {
			value:    nil,
			expected: nil,
		},
		"string": {
			value:    "test error",
			expected: errors.New("test error"),
		},
		"error": {
			value:    errors.New("test error"),
			expected: errors.New("test error"),
		},
		"recover type proxy": {
			value:    365,
			expected: errors.New("panic: 365"),
		},
		"nil w/ underlying": {
			value:      nil,
			expected:   errors.New("underlying"),
			underlying: errors.New("underlying"),
		},
		"string w/ underlying": {
			value:      "test error",
			expected:   errors.New("test error"),
			underlying: errors.New("underlying"),
		},
		"error w/ underlying": {
			value:      errors.New("test error"),
			expected:   errors.New("test error"),
			underlying: errors.New("underlying"),
		},
		"recover type proxy w/ underlying": {
			value:      365,
			expected:   errors.New("panic: 365"),
			underlying: errors.New("underlying"),
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			err := recoverErr(test.underlying, test.value)
			if err == nil && test.expected == nil {
				return
			}

			if err.Error() != test.expected.Error() {
				t.Errorf(
					"expected %s, got %s",
					test.expected.Error(),
					err.Error(),
				)
			}

			under, ok := err.(wrappedError)
			if !ok {
				if test.underlying != nil && test.value != nil {
					t.Fatalf(
						"expected %s, got %s",
						test.underlying.Error(),
						err.Error(),
					)
				}

				return
			}

			if under.Unwrap() != test.underlying {
				t.Fatalf(
					"expected %s, got %s",
					test.underlying.Error(),
					under.Unwrap().Error(),
				)
			}
		})
	}
}

func Test_Publisher_ReadEvents_NegBuff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	events := publisher.ReadEvents(-1)
	timer := time.NewTimer(time.Millisecond)

	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case <-timer.C:
	case publisher.events <- &testEvent{msg: "test"}:
		t.Fatal("event should not be published")
	case <-events:
		t.Fatal("expected no events")
	}
}

func Test_Publisher_ReadErrors_NegBuff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	errs := publisher.ReadErrors(-1)
	timer := time.NewTimer(time.Millisecond)

	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case <-timer.C:
	case publisher.errors <- errors.New("test error"):
		t.Fatal("event should not be published")
	case <-errs:
		t.Fatal("expected no events")
	}
}

func Test_Publisher_EventFunc_NilEventsChan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	err := publisher.EventFunc(ctx, func() Event {
		return nil
	})

	if publisher.events == nil && err != nil {
		t.Fatal("unexpected error")
	}
}

func Test_Publisher_ErrorFunc_NilErrorChan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	err := publisher.ErrorFunc(ctx, func() error {
		return nil
	})

	if publisher.errors == nil && err != nil {
		t.Fatal("unexpected error")
	}
}

type ctxtest struct {
	parent context.Context
	child  context.Context
}

func ctxCancelTests() map[string]ctxtest {
	ctx := context.Background()

	cancelledCtx, cancelledCancel := context.WithCancel(context.Background())
	cancelledCancel()

	return map[string]ctxtest{
		"child cancel": {
			parent: ctx,
			child:  cancelledCtx,
		},
		"parent cancel": {
			parent: cancelledCtx,
			child:  ctx,
		},
	}
}

func Test_ErrorFunc_CtxCancel(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			publisher := NewPublisher(test.parent)
			defer func() {
				err := publisher.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			_ = publisher.ReadErrors(0)

			err := publisher.ErrorFunc(test.child, func() error {
				return nil
			})

			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func Test_EventFunc_CtxCancel(t *testing.T) {
	for name, test := range ctxCancelTests() {
		t.Run(name, func(t *testing.T) {
			publisher := NewPublisher(test.parent)
			defer func() {
				err := publisher.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			_ = publisher.ReadEvents(0)

			err := publisher.EventFunc(test.child, func() Event {
				return nil
			})

			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func Test_Publisher_EventFunc(t *testing.T) {
	testdata := map[string]struct {
		efunc    EventFunc
		expected string
		err      bool
	}{
		"single event": {
			EventFunc(func() Event {
				return &testEvent{msg: "test"}
			}),
			"test",
			false,
		},
		"panic event func": {
			EventFunc(func() Event {
				panic("test panic")
			}),
			"",
			true,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			publisher := NewPublisher(ctx)
			defer func() {
				err := publisher.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			events := publisher.ReadEvents(1)

			err := publisher.EventFunc(ctx, test.efunc)

			if err != nil {
				if !test.err {
					t.Fatal("expected error")
				}

				return
			}

			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case e, ok := <-events:
				if !ok {
					t.Fatal("expected event")
				}

				if e.Event() != test.expected {
					t.Fatalf(
						"expected %q, got %q",
						test.expected,
						e.Event(),
					)
				}
			}
		})
	}
}

func Test_Publisher_ErrorFunc(t *testing.T) {
	testdata := map[string]struct {
		efunc    ErrorFunc
		expected string
		err      bool
	}{
		"single error": {
			ErrorFunc(func() error {
				return errors.New("test")
			}),
			"test",
			false,
		},
		"panic error func": {
			ErrorFunc(func() error {
				panic("test panic")
			}),
			"",
			true,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			publisher := NewPublisher(ctx)
			defer func() {
				err := publisher.Close()
				if err != nil {
					t.Errorf("Publisher.Close() failed: %v", err)
				}
			}()

			errs := publisher.ReadErrors(1)

			err := publisher.ErrorFunc(ctx, test.efunc)

			if err != nil {
				if !test.err {
					t.Fatal("expected error")
				}

				return
			}

			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case e, ok := <-errs:
				if !ok {
					t.Fatal("expected event")
				}

				if e.Error() != test.expected {
					t.Fatalf(
						"expected %q, got %q",
						test.expected,
						e.Error(),
					)
				}
			}
		})
	}
}

func testErrorStreams(
	ctx context.Context,
	streamcount int,
) (streams []ErrorStream, errs int) {
	streams = make([]ErrorStream, streamcount+1)

	for i := 0; i < streamcount; i++ {
		// Make the last one nil
		if i == streamcount {
			streams[i] = nil
		}

		s, size := testErrorStream(ctx)
		errs += size
		streams[i] = s
	}

	return streams, errs
}

// nolint:gocritic
func testErrorStream(ctx context.Context) (ErrorStream, int) {
	errs := make(chan error)
	size := rand.Int() % 100

	go func(errs ErrorWriter, size int) {
		defer close(errs)

		for i := 0; i < size; i++ {
			select {
			case <-ctx.Done():
				return
			case errs <- errors.New("test"):
			}
		}
	}(errs, size)

	return errs, size
}

func Test_Publisher_Errors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streams, errcount := testErrorStreams(ctx, 10)
	if errcount == 0 {
		t.Fatal("expected error")
	}

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	errs := publisher.ReadErrors(errcount)
	publisher.Errors(ctx, streams...)

	for i := 0; i < errcount; i++ {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case err, ok := <-errs:
			if !ok || err == nil {
				t.Fatal("expected error")
			}
		}
	}
}

func Test_Publisher_Errors_NilErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	err := publisher.Errors(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func testEventStreams(
	ctx context.Context,
	streamcount int,
) (streams []EventStream, events int) {
	streams = make([]EventStream, streamcount+1)

	for i := 0; i < streamcount; i++ {
		// Make the last one nil
		if i == streamcount {
			streams[i] = nil
		}

		s, size := testEventStream(ctx)
		events += size
		streams[i] = s
	}

	return streams, events
}

// nolint:gocritic
func testEventStream(ctx context.Context) (EventStream, int) {
	events := make(chan Event)
	size := rand.Int() % 100

	go func(events EventWriter, size int) {
		defer close(events)

		for i := 0; i < size; i++ {
			select {
			case <-ctx.Done():
				return
			case events <- &testEvent{"test"}:
			}
		}
	}(events, size)

	return events, size
}

func Test_Publisher_Events(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streams, eventcount := testEventStreams(ctx, 10)
	if eventcount == 0 {
		t.Fatal("expected events")
	}

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	events := publisher.ReadEvents(eventcount)
	publisher.Events(ctx, streams...)

	for i := 0; i < eventcount; i++ {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case err, ok := <-events:
			if !ok || err == nil {
				t.Fatal("expected error")
			}
		}
	}
}

func Test_Publisher_Events_NilEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	err := publisher.Events(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func Test_Publisher_Split_NilInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	err := publisher.Split(ctx, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func Test_Publisher_Split_NilEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	in := make(chan interface{})
	defer close(in)

	err := publisher.Split(ctx, in)
	if err == nil {
		t.Fatal("expected error")
	}
}

func Test_Publisher_Split_NilErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	// Init events to bypass nil check
	_ = publisher.ReadEvents(0)

	in := make(chan interface{})
	defer close(in)

	err := publisher.Split(ctx, in)
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "no listener for errors" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// nolint:goconst
func Test_Publisher_Split_TypeSwitch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := NewPublisher(ctx)
	defer func() {
		err := publisher.Close()
		if err != nil {
			t.Errorf("Publisher.Close() failed: %v", err)
		}
	}()

	in := make(chan interface{}, 4)
	defer close(in)

	in <- nil
	in <- &testEvent{"test"}
	in <- errors.New("test")
	in <- &testCombo{"test"}

	events := publisher.ReadEvents(1)
	errs := publisher.ReadErrors(2)

	err := publisher.Split(ctx, in)
	if err != nil {
		t.Fatal("expected success")
	}

	event, ok := <-events
	if !ok || event == nil {
		t.Fatal("expected event")
	}

	err, ok = <-errs
	if !ok || err == nil {
		t.Fatal("expected error")
	}

	err, ok = <-errs
	if !ok || err == nil {
		t.Fatal("expected error")
	}

	combo, ok := err.(Combo)
	if !ok {
		t.Fatal("expected Combo")
	}

	if combo.Event() != "test" || combo.Error() != "test" {
		t.Fatal("expected test")
	}
}

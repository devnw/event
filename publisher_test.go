package event

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testEvent struct{ msg string }

func (e *testEvent) Event() string {
	return e.msg
}

// nolint:unused
type testCombo struct{ msg string }

// nolint:unused
func (c *testCombo) Error() error {
	return errors.New(c.msg)
}

// nolint:unused
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

func Test_recoverErr(t *testing.T) {
	testdata := map[string]struct {
		value    interface{}
		expected error
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
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			err := recoverErr(test.value)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cancelledCtx, cancelledCancel := context.WithCancel(ctx)
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

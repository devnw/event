// Package event provides access to a publisher
// for events and errors which implement the `error` or Event
// types. This package is intended to be used by highly
// concurrent implementations which require streams of events
// and errors rather than one-off events and errors.
//
// Creation of Events/errors can happen using several different
// methods. The most common use case will likely be reading from
// an Event/ErrorStream that the publisher subscribes to and is
// supplied to the `Publisher.Events/Errors` method.
//
// Alternatively, the `Publisher.EventFunc/ErrorFunc` methods
// provide support for deferred creation and rendering of event
// and error data for publishing. This allows for a reduction in
// memory allocations in the event that there are no subscribers
// for events or errors respectively.
//
// Once a publisher is instantiated using the `NewPublisher` method,
// a user must create subscriptions to the publisher's events or errors
// using the `ReadErrors/ReadEvents` methods prior to publishing otherwise
// the events and errors will be lost and the publish methods will error.
package event

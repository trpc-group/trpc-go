package rpcz

import "time"

// Event describes something happened at specific timestamp.
type Event struct {
	// Name of Event.
	Name string
	// Time of Event happened.
	Time time.Time
}

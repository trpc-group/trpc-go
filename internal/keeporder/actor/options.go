package actor

import "time"

// Options specifies the actor's options.
type Options struct {
	IdleGroupTimeout time.Duration
	MaxElementCount  int
}

func (o *Options) fixDefault() {
	if o.IdleGroupTimeout == 0 {
		o.IdleGroupTimeout = defaultIdleGroupTimeout
	}
	if o.MaxElementCount == 0 {
		o.MaxElementCount = defaultMaxElementCount
	}
}

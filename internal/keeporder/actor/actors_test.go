package actor_test

import (
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/internal/keeporder/actor"
)

func TestActors(t *testing.T) {
	actors := actor.NewActors()
	key := "testKey"
	called := false

	// Define a function to be added.
	fn := func() {
		called = true
	}

	// Add the function to the actors under the specified key.
	actors.Add(key, fn)

	// Allow some time for the actor to process the function.
	time.Sleep(100 * time.Millisecond)

	// Check if the function was called.
	if !called {
		t.Errorf("The function was not called.")
	}

	// Remove the actor from the actors.
	actors.Remove(key)

	// Stop all the actors.
	// This should not cause panic when executed after remove.
	actors.Stop()
}

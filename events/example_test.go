package events_test

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/events"
)

// ExamplePublish shows the non-hook core: subscribe to a typed topic, publish to
// it, and unsubscribe when done. Inside a component prefer events.UseTopic, which
// ties the subscription to the component lifecycle.
func ExamplePublish() {
	parseUnsubscribe := events.Subscribe("greeting", func(parseValue string) {
		fmt.Println("received:", parseValue)
	})
	defer parseUnsubscribe()

	events.Publish("greeting", "hello")
	events.Publish("greeting", "world")
	// Output:
	// received: hello
	// received: world
}

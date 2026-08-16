package desktop

import "testing"

func TestRuntimeSubscribeAndUnsubscribe(t *testing.T) {
	runtime := NewRuntime()
	var names []string

	unsubscribe := runtime.Subscribe(func(name string, _ any) {
		names = append(names, name)
	})
	runtime.Emit("first", nil)
	unsubscribe()
	runtime.Emit("second", nil)

	if len(names) != 1 || names[0] != "first" {
		t.Fatalf("listener events = %v, want [first]", names)
	}
}

func TestRuntimeIgnoresNilListener(t *testing.T) {
	runtime := NewRuntime()
	unsubscribe := runtime.Subscribe(nil)
	unsubscribe()
	runtime.Emit("event", nil)
}

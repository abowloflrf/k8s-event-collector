package receiver

import corev1 "k8s.io/api/core/v1"

// Receiver interface must be implemented by any receiver
type Receiver interface {
	// Name of this receiver
	Name() string
	// Send event to target
	Send(e *corev1.Event) error
	// Filter before Send, return false if the event should be ignored
	Filter(e *corev1.Event) bool
	// Close the receiver if needed, eg. opened file, tcp connection
	Close()
}

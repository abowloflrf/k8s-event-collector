package receiver

import corev1 "k8s.io/api/core/v1"

type Receiver interface {
	Send(e *corev1.Event) error
	Close()
}

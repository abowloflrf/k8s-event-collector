package receiver

import corev1 "k8s.io/api/core/v1"

type DiscardTarget struct {
}

func NewDiscardTarget() (*DiscardTarget, error) {
	return &DiscardTarget{}, nil
}

func (dt *DiscardTarget) Send(*corev1.Event) error {
	return nil
}

func (dt *DiscardTarget) Close() {

}

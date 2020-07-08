package receiver

import corev1 "k8s.io/api/core/v1"

type DiscardTarget struct {
}

func NewDiscardTarget() (*DiscardTarget, error) {
	return &DiscardTarget{}, nil
}

func (dt *DiscardTarget) Name() string {
	return "blackhole"
}

func (dt *DiscardTarget) Send(*corev1.Event) error {
	return nil
}

func (dt *DiscardTarget) Filter(e *corev1.Event) bool {
	return true
}

func (dt *DiscardTarget) Close() {

}

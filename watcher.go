package main

import (
	"time"

	"github.com/abowloflrf/k8s-events-dispatcher/config"
	receiver "github.com/abowloflrf/k8s-events-dispatcher/receivers"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type EventController struct {
	clientset         kubernetes.Interface
	informerFactory   informers.SharedInformerFactory
	evInformer        coreinformers.EventInformer
	stopper           chan struct{}
	informerHasSynced bool
	r                 receiver.Receiver
}

func (ec *EventController) handleEvent(e *corev1.Event, handleType string) {
	logrus.Infof("event %s [%s][%s][%s][%s], created since %v", handleType, string(e.UID), e.Namespace, e.InvolvedObject.Name, e.Reason, time.Now().Sub(e.CreationTimestamp.Time))
	// prevent handle the old events when controller just start
	if time.Now().Sub(e.LastTimestamp.Time) > time.Second*5 {
		return
	}

	logrus.Infof("event to send [%s]", string(e.UID))
	err := ec.r.Send(e)
	if err != nil {
		logrus.Errorf("send event error: %v", err)
	}
}

func (ec *EventController) Run(stop <-chan struct{}) {
	logrus.Info("starting event controller")
	ec.informerFactory.Start(stop)
	if !cache.WaitForCacheSync(stop, ec.evInformer.Informer().HasSynced) {
		logrus.Error("wait for cache sync error")
		return
	}
	ec.informerHasSynced = true

	logrus.Info("informer cache synced, controller started")
	<-stop
}

func (ec *EventController) addHandlers() {
	ec.evInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event := obj.(*corev1.Event)
			ec.handleEvent(event, "ADD")
		},
		UpdateFunc: func(_, newObj interface{}) {
			event := newObj.(*corev1.Event)
			ec.handleEvent(event, "UPDATE")
		},
		DeleteFunc: func(obj interface{}) {
			// Nothing to do
		},
	})
}

func (ec *EventController) Stop() {
	close(ec.stopper)
}

func NewEventController(cs *kubernetes.Clientset) *EventController {
	factory := informers.NewSharedInformerFactory(cs, 0)
	evInformer := factory.Core().V1().Events()

	// only es receiver was implemented currently
	var target receiver.Receiver
	escfg := config.C.Receivers.ElasticSearch
	target, err := receiver.NewElasticsearchTarget(escfg.Addresses, escfg.Index)
	if err != nil {
		logrus.Errorf("create receiver error: %v", err)
		target, _ = receiver.NewDiscardTarget()
	}

	ec := &EventController{
		clientset:       cs,
		informerFactory: factory,
		evInformer:      evInformer,

		stopper: make(chan struct{}),
		r:       target,
	}
	ec.addHandlers()

	return ec
}

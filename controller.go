package main

import (
	"time"

	"github.com/abowloflrf/k8s-events-dispatcher/config"
	"github.com/abowloflrf/k8s-events-dispatcher/receiver"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type EventController struct {
	clientset         kubernetes.Interface
	informerFactory   informers.SharedInformerFactory
	evInformer        coreinformers.EventInformer
	stopper           chan struct{}
	informerHasSynced bool
	r                 receiver.Receiver
	queue             workqueue.Interface
}

func (ec *EventController) enqueue(e *corev1.Event, handleType string) {
	logrus.Infof("event %s [%s][%s/%s][%s], last since %v", handleType, string(e.UID), e.InvolvedObject.Namespace, e.InvolvedObject.Name, e.Reason, time.Since(e.LastTimestamp.Time))
	// prevent handle the old events when controller just start
	if time.Since(e.LastTimestamp.Time) > time.Second*5 {
		return
	}
	logrus.Infof("event to send [%s]", string(e.UID))
	ec.queue.Add(e)
}

func (ec *EventController) worker() {
	for ec.processNextItem() {
	}
}

func (ec *EventController) processNextItem() bool {
	item, quit := ec.queue.Get()
	if quit {
		return false
	}
	event := item.(*corev1.Event)
	defer ec.queue.Done(item)
	err := ec.r.Send(event)
	if err != nil {
		logrus.Errorf("send event error: %v", err)
	}
	return true
}

func (ec *EventController) Run(workers int, stop <-chan struct{}) {
	logrus.Info("starting event controller")
	defer logrus.Info("stopping event controller")
	defer ec.queue.ShutDown()
	ec.informerFactory.Start(stop)
	if !cache.WaitForCacheSync(stop, ec.evInformer.Informer().HasSynced) {
		logrus.Error("wait for cache sync error")
		return
	}
	ec.informerHasSynced = true
	logrus.Info("informer cache synced, controller started")
	for i := 0; i < workers; i++ {
		go wait.Until(ec.worker, time.Second, stop)
	}
	<-stop
}

func (ec *EventController) addHandlers() {
	ec.evInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			e := obj.(*corev1.Event)
			ec.enqueue(e, "ADD")
		},
		UpdateFunc: func(_, newObj interface{}) {
			e := newObj.(*corev1.Event)
			ec.enqueue(e, "UPDATE")
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
	target, err := receiver.NewElasticsearchTarget(config.C.Receivers.ElasticSearch)
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
		queue:   workqueue.New(),
	}
	ec.addHandlers()

	return ec
}

package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func getKubeClient() (*kubernetes.Clientset, error) {
	var c *rest.Config
	c, err := rest.InClusterConfig()
	if err != nil && err == rest.ErrNotInCluster {
		c, err = clientcmd.BuildConfigFromFlags("", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(c)
}

type EventController struct {
	clientset       kubernetes.Interface
	informerFactory informers.SharedInformerFactory
	evInformer      coreinformers.EventInformer
	stopper         chan struct{}
}

func (ec *EventController) handleEvent(e *corev1.Event) {
	// Type: (Normal, Warning)
	log.Println(e.Namespace, e.Type, e.Reason, e.ObjectMeta.Name, e.Message)
}

func (ec *EventController) Start(stop <-chan struct{}) {
	log.Println("starting event controller")
	ec.informerFactory.Start(stop)
	if !cache.WaitForCacheSync(stop, ec.evInformer.Informer().HasSynced) {
		log.Println("wait for cache sync error")
		return
	}
	<-stop
}

func (ec *EventController) Stop() {
	close(ec.stopper)
}

func NewEventController(cs *kubernetes.Clientset) *EventController {
	factory := informers.NewSharedInformerFactory(cs, 0)

	evInformer := factory.Core().V1().Events()
	ec := &EventController{
		clientset:       cs,
		informerFactory: factory,
		evInformer:      evInformer,

		stopper: make(chan struct{}),
	}
	evInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event := obj.(*corev1.Event)
			ec.handleEvent(event)
		},
		UpdateFunc: func(_, newObj interface{}) {
			event := newObj.(*corev1.Event)
			ec.handleEvent(event)
		},
		DeleteFunc: func(obj interface{}) {
			// Nothing to do
		},
	})
	return ec
}

func main() {
	cs, err := getKubeClient()
	if err != nil {
		log.Fatal(err)
	}

	ec := NewEventController(cs)
	stop := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Println("receive signal", sig)
		close(stop)
	}()
	ec.Start(stop)
	log.Println("exited")
}

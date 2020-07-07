package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/abowloflrf/k8s-events-dispatcher/config"
	"github.com/sirupsen/logrus"
)

var configFile string
var LeaderElect bool

const component = "eventsdispatcher"

func init() {
	flag.StringVar(&configFile, "c", "", "config file to use, default: /etc/eventsdispatcher/config.json")
	flag.BoolVar(&LeaderElect, "leaderelect", false, "set true to enable leader election mode, by default use standalone mode")
	flag.Parse()
	// initial logger using logurs
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	})
	// initial configuration using viper
	config.InitConf(configFile)
}

func main() {
	cs, err := getKubeClient()
	if err != nil {
		logrus.Fatal(err)
	}

	stop := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		sig := <-sigs
		logrus.Println("receive signal", sig)
		close(stop)
	}()

	run := func(ctx context.Context) {
		ec := NewEventController(cs)
		ec.Run(stop)
	}

	// standalone mode
	if !LeaderElect {
		logrus.Info("starting with stand-alone mode")
		run(ctx)
		logrus.Fatal("unreachable")
	}

	// leader-election mode
	var id string
	var leaseLockName = component
	var leaseLockNamespace string
	if !config.C.LeaderElect {
		logrus.Println("exited")
		os.Exit(0)
	}
	id, err = os.Hostname()
	if err != nil {
		logrus.Fatalf("get hostname %v", err)
	}
	id = id + "_" + string(uuid.NewUUID())
	leaseLockNamespace, err = getInClusterNamespace()
	if err != nil {
		leaseLockNamespace = corev1.NamespaceDefault
	}
	logrus.Infof("starting with leader-election mode, ID: %s", id)
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Namespace: leaseLockNamespace,
			Name:      leaseLockName,
		},
		Client: cs.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logrus.Infof("start leading: %s", id)
				run(ctx)
			},
			OnStoppedLeading: func() {
				logrus.Fatalf("leaderelection lost: %s", id)
			},
		},
		ReleaseOnCancel: true,
		Name:            component,
	})
	logrus.Println("exited")
}

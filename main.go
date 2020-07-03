package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abowloflrf/k8s-events-dispatcher/config"
	"github.com/sirupsen/logrus"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "t", "", "config file to use, default: /etc/eventsdispatcher/config.json")
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

	ec := NewEventController(cs)
	stop := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logrus.Println("receive signal", sig)
		close(stop)
	}()
	ec.Run(stop)
	logrus.Println("exited")
}

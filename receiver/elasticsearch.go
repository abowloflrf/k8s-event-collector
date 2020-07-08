package receiver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/abowloflrf/k8s-event-collector/config"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

type ElasticsearchTarget struct {
	client *elasticsearch.Client
	config elasticsearch.Config
	index  string
}

func NewElasticsearchTarget(cfg *config.ElasticSearch) (*ElasticsearchTarget, error) {
	escfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	}
	c, err := elasticsearch.NewClient(escfg)
	if err != nil {
		return nil, err
	}

	return &ElasticsearchTarget{
		client: c,
		config: escfg,
		index:  cfg.Index,
	}, nil
}

func (et *ElasticsearchTarget) Name() string {
	return "elasticsearch"
}

func (et *ElasticsearchTarget) Send(e *corev1.Event) error {
	toSend, err := json.Marshal(e)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Body:       bytes.NewBuffer(toSend),
		Index:      et.index,
		DocumentID: string(e.UID),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := req.Do(ctx, et.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	if resp.HasWarnings() {
		logrus.Warningf("request to elasticsearch: %s", resp.Warnings())
	}
	if resp.IsError() {
		return fmt.Errorf("request to elasticsearch error, status: %s resp: %s", resp.Status(), resp.String())
	}
	return nil
}

func (et *ElasticsearchTarget) Filter(e *corev1.Event) bool {
	return true
}

func (et *ElasticsearchTarget) Close() {}

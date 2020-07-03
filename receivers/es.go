package receiver

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	corev1 "k8s.io/api/core/v1"
)

type EsTarget struct {
	client *elasticsearch.Client
	config elasticsearch.Config
	index  string
}

func NewElasticsearchTarget(address []string, index string) (*EsTarget, error) {
	escfg := elasticsearch.Config{
		Addresses: address,
	}
	c, err := elasticsearch.NewClient(escfg)
	if err != nil {
		return nil, err
	}

	return &EsTarget{
		client: c,
		config: escfg,
		index:  index,
	}, nil
}

func (et *EsTarget) Send(e *corev1.Event) error {
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
	_ = resp.Body
	return nil
}

func (et *EsTarget) Close() {}
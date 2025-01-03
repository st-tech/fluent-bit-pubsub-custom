package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
)

type Keeper interface {
	Send(ctx context.Context, data []byte, attributes map[string]string) *pubsub.PublishResult
	Stop()
}

type GooglePubSub struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

func NewKeeper(projectId, topicName string,
	publishSetting *pubsub.PublishSettings) (Keeper, error) {
	if projectId == "" || topicName == "" {
		return nil, fmt.Errorf("[err] NewKeeper empty params")
	}
	ctx := context.Background()

	// ADC based authentication
	// https://cloud.google.com/docs/authentication/application-default-credentials
	client, err := pubsub.NewClient(ctx, projectId)

	if err != nil {
		return nil, errors.Wrap(err, "[err] pubsub client")
	}

	topic := client.Topic(topicName)

	if publishSetting != nil {
		topic.PublishSettings = *publishSetting
	} else {
		topic.PublishSettings = pubsub.DefaultPublishSettings
	}

	pubs := &GooglePubSub{client: client, topic: topic}
	return Keeper(pubs), nil
}

func (gps *GooglePubSub) Send(ctx context.Context, data []byte, attributes map[string]string) *pubsub.PublishResult {
	if len(data) == 0 {
		return nil
	}
	msg := &pubsub.Message{Data: data}
	if attributes != nil {
		msg.Attributes = attributes
	}
	return gps.topic.Publish(ctx, msg)
}

func (gps *GooglePubSub) Stop() {
	gps.topic.Stop()
}

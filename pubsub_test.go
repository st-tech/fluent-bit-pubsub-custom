package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestNewKeeper(t *testing.T) {
	assert := assert.New(t)

	err := godotenv.Load()
	if err != nil {
		t.Log("Error loading .env file")
	}

	_, err = NewKeeper("", "", "", nil)
	assert.Error(err)

	projectId := os.Getenv("PROJECT_ID")
	topicName := os.Getenv("TOPIC_NAME")
	region := os.Getenv("REGION")

	if projectId == "" || topicName == "" {
		return
	}

	_, err = NewKeeper(projectId, topicName, "", nil)
	assert.NoError(err)

	keeper, err := NewKeeper(projectId, topicName, region, &pubsub.PublishSettings{
		ByteThreshold:  10,
		CountThreshold: 10,
		DelayThreshold: 1 * time.Second,
		Timeout:        5 * time.Second,
	})
	assert.NoError(err)
	assert.Equal(keeper.(*GooglePubSub).topic.PublishSettings.Timeout, 5*time.Second)
	assert.Equal(keeper.(*GooglePubSub).topic.PublishSettings.DelayThreshold, 1*time.Second)
	assert.Equal(keeper.(*GooglePubSub).topic.PublishSettings.CountThreshold, 10)
	assert.Equal(keeper.(*GooglePubSub).topic.PublishSettings.ByteThreshold, 10)
}

func TestGooglePubSub_Send(t *testing.T) {
	assert := assert.New(t)

	err := godotenv.Load()
	if err != nil {
		t.Log("Error loading .env file")
	}

	projectId := os.Getenv("PROJECT_ID")
	topicName := os.Getenv("TOPIC_NAME")

	if projectId == "" || topicName == "" {
		return
	}

	ctx := context.Background()
	keeper, err := NewKeeper(projectId, topicName, "", nil)
	assert.NoError(err)

	data := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	attributes := map[string]string{
		"attr_key1": "attr_value1",
		"attr_key2": "attr_value2",
	}
	msg, err := json.Marshal(data)
	assert.NoError(err)
	result := keeper.Send(ctx, msg, attributes)

	_, err = result.Get(ctx)
	assert.NoError(err)
	sub := keeper.(*GooglePubSub).client.Subscription(topicName)
	go func() {
		sub.Receive(context.Background(), func(ctx context.Context, m *pubsub.Message) {
			log.Printf("Got message: %s", m.Data)
			m.Ack()
		})
	}()
	time.Sleep(5 * time.Second)
}

func TestGooglePubSub_Stop(t *testing.T) {
	assert := assert.New(t)

	err := godotenv.Load()
	if err != nil {
		t.Log("Error loading .env file")
	}

	projectId := os.Getenv("PROJECT_ID")
	topicName := os.Getenv("TOPIC_NAME")
	if projectId == "" || topicName == "" {
		return
	}

	keeper, err := NewKeeper(projectId, topicName, "", nil)
	assert.NoError(err)
	keeper.Stop()
}

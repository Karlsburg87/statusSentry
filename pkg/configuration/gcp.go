package configuration

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/pubsub"
)

//Publisher publishes to Google Cloud Platform PubSub
func Publisher(stuff <-chan Transporter) {
	//GCP config
	projectID := os.Getenv("PROJECT_ID")
	//setup topic names and defaults
	pingTopic := os.Getenv("PING_RESPONSE_TOPIC")
	if pingTopic == "" {
		pingTopic = "pagePings"
	}
	statusUpdateTopic := os.Getenv("STATUS_UPDATE_TOPIC")
	if statusUpdateTopic == "" {
		statusUpdateTopic = "statusUpdates"
	}

	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)

	if err != nil {
		log.Panicf("pubsub.NewClient: %v", err)
	}
	defer client.Close()

	//create topics if not exist
	var topic, statusT, pingT *pubsub.Topic
	if pingT, err = client.CreateTopic(ctx, pingTopic); err != nil {
		//assume Topic exists so get
		pingT = client.Topic(pingTopic)

	}
	if statusT, err = client.CreateTopic(ctx, statusUpdateTopic); err != nil {
		log.Printf("received error when creating topic - assuming topic exists: %v", err)
		//assume Topic exists so get
		statusT = client.Topic(statusUpdateTopic)
	}
	log.Printf("GCP PubSub goroutine ready to receive using topics %s and %s", pingTopic, statusUpdateTopic)
	for msg := range stuff {
		//different topics for ping and status page updates
		// Hardcoded by design
		topic = statusT
		if msg.PingResponse != nil {
			topic = pingT
		}
		payload, err := msg.ToJSON()
		if err != nil {
			log.Printf("error in PubSub in JSON convert: %v\n", err)
		}
		_ = topic.Publish(ctx, &pubsub.Message{
			Data: payload,
		})
		// No blocking Get. See: https://pkg.go.dev/cloud.google.com/go/pubsub#hdr-Publishing
	}
}

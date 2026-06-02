package internal

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

const KafkaTopic = "burrow.tunnel-stats"

type TunnelStatsEvent struct {
	TunnelID  string    `json:"tunnel_id"`
	ClientIP  string    `json:"client_ip"`
	UserID    *int64    `json:"user_id,omitempty"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Duration  string    `json:"duration"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

func NewKafkaProducer(brokers []string) (sarama.AsyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}
	return producer, nil
}

func EnsureTopic(brokers []string) error {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	admin, err := sarama.NewClusterAdmin(brokers, config)
	if err != nil {
		return fmt.Errorf("create kafka admin: %w", err)
	}
	defer admin.Close()

	detail := &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}
	return admin.CreateTopic(KafkaTopic, detail, false)
}

func PublishEvent(producer sarama.AsyncProducer, event *TunnelStatsEvent) error {
	b, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	msg := &sarama.ProducerMessage{
		Topic: KafkaTopic,
		Key:   sarama.StringEncoder(event.TunnelID),
		Value: sarama.ByteEncoder(b),
	}
	producer.Input() <- msg
	return nil
}
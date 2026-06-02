package internal

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/IBM/sarama"
)

type StatsConsumer struct {
	brokers []string
	groupID string
	pg      *PostgresClient
}

func NewStatsConsumer(brokers []string, groupID string, pg *PostgresClient) *StatsConsumer {
	return &StatsConsumer{
		brokers: brokers,
		groupID: groupID,
		pg:      pg,
	}
}

func (c *StatsConsumer) Run(ctx context.Context) error {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumerGroup, err := sarama.NewConsumerGroup(c.brokers, c.groupID, config)
	if err != nil {
		return err
	}
	defer consumerGroup.Close()

	handler := &consumerHandler{pg: c.pg}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := consumerGroup.Consume(ctx, []string{KafkaTopic}, handler); err != nil {
				log.Printf("kafka consumer error: %v", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

type consumerHandler struct {
	pg *PostgresClient
}

func (h *consumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var event TunnelStatsEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("kafka: unmarshal event: %v", err)
			session.MarkMessage(msg, "")
			continue
		}

		bucketStart := event.Timestamp.Truncate(5 * time.Minute)
		if err := h.pg.UpsertTunnelStat(session.Context(), event.TunnelID, bucketStart, event.Size, 0, 1); err != nil {
			log.Printf("kafka: upsert stat: %v", err)
		}

		logEntry := &TunnelLog{
			TunnelID:   event.TunnelID,
			Timestamp:  event.Timestamp,
			Method:     event.Method,
			Path:       event.Path,
			StatusCode: 0,
			Size:       event.Size,
			Duration:   event.Duration,
			IP:         event.IP,
		}
		if err := h.pg.InsertTunnelLog(session.Context(), event.TunnelID, logEntry); err != nil {
			log.Printf("kafka: insert log: %v", err)
		}

		session.MarkMessage(msg, "")
	}
	return nil
}
package broker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Event struct {
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokerAddr, topic string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddr),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &KafkaProducer{
		writer: writer,
	}
}

func (p *KafkaProducer) PublishProductCreated(ctx context.Context, product interface{}) error {
	event := Event{
		EventType: "ProductCreated",
		Timestamp: time.Now(),
		Payload:   product,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(time.Now().String()),
			Value: data,
		},
	)

	if err != nil {
		log.Printf("❌ Ошибка отправки сообщения в Kafka: %v", err)
		return err
	}

	log.Printf("📨 Событие ProductCreated успешно отправлено в Kafka")
	return nil
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

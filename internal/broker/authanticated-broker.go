package broker

import (
	"context"
	"errors"
	"time"

	"github.com/sanke08/in-mem-message-queue/internal/auth"
	"github.com/sanke08/in-mem-message-queue/internal/queue"
)

var ErrUnauthorized = errors.New("unauthorized: invalid API key")

type AuthenticatedBroker struct {
	broker *Broker
	Auth   *auth.AuthManager
}

func NewAuthenticatedBroker(b *Broker, a *auth.AuthManager) *AuthenticatedBroker {
	return &AuthenticatedBroker{
		broker: b,
		Auth:   a,
	}
}

func (ab *AuthenticatedBroker) ValidateAndGetQueueName(ctx context.Context, apiKey string, queueName string) (string, error) {
	_, tenantID, ok := ab.Auth.ValidateKey(ctx, apiKey)

	if !ok {
		return "", ErrUnauthorized
	}

	fullQueueName := tenantID + ":" + queueName
	return fullQueueName, nil
}

func (ab *AuthenticatedBroker) Publish(ctx context.Context, apiKey string, queueName string, payload []byte) (string, error) {
	fullQueueName, err := ab.ValidateAndGetQueueName(ctx, apiKey, queueName)
	if err != nil {
		return "", err
	}
	return ab.broker.Publish(fullQueueName, payload)
}

func (ab *AuthenticatedBroker) Claim(ctx context.Context, apiKey string, queueName string, waitSeconds int) (*queue.Message, error) {
	fullQueueName, err := ab.ValidateAndGetQueueName(ctx, apiKey, queueName)
	if err != nil {
		return nil, err
	}
	return ab.broker.Claim(fullQueueName, time.Duration(waitSeconds)*time.Second)
}

func (ab *AuthenticatedBroker) Ack(ctx context.Context, apiKey string, queueName string, messageID string) error {
	fullQueueName, err := ab.ValidateAndGetQueueName(ctx, apiKey, queueName)
	if err != nil {
		return err
	}
	return ab.broker.Ack(fullQueueName, messageID)
}

func (ab *AuthenticatedBroker) Stats(ctx context.Context, apiKey string, queueName string) (queue.QueueStats, error) {
	fullQueueName, err := ab.ValidateAndGetQueueName(ctx, apiKey, queueName)
	if err != nil {
		return queue.QueueStats{}, ErrUnauthorized
	}
	return ab.broker.Stats(fullQueueName), nil
}

package queue

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
)

// SQSConfig holds settings for LocalStack or real AWS SQS.
type SQSConfig struct {
	Endpoint  string // e.g. http://localhost:4566 (required)
	Region    string
	QueueName string
	AccessKey string
	SecretKey string
}

// SQSClient implements Publisher and can receive messages for the worker.
type SQSClient struct {
	api      *sqs.Client
	queueURL string
}

// NewSQS builds an SQS client, ensures the queue exists, and returns the URL.
func NewSQS(ctx context.Context, cfg SQSConfig) (*SQSClient, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("queue: SQS endpoint is empty")
	}
	if cfg.QueueName == "" {
		return nil, fmt.Errorf("queue: SQS queue name is empty")
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	ak := cfg.AccessKey
	if ak == "" {
		ak = "test"
	}
	sk := cfg.SecretKey
	if sk == "" {
		sk = "test"
	}

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(ak, sk, "")),
	)
	if err != nil {
		return nil, err
	}

	api := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	})

	url, err := ensureQueueURL(ctx, api, cfg.QueueName)
	if err != nil {
		return nil, err
	}
	return &SQSClient{api: api, queueURL: url}, nil
}

func ensureQueueURL(ctx context.Context, api *sqs.Client, name string) (string, error) {
	gout, err := api.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(name)})
	if err == nil {
		return aws.ToString(gout.QueueUrl), nil
	}
	_, cerr := api.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(name)})
	if cerr != nil && !isQueueExistsErr(cerr) {
		return "", fmt.Errorf("create queue %q: %w", name, cerr)
	}
	gout, err = api.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(name)})
	if err != nil {
		return "", err
	}
	return aws.ToString(gout.QueueUrl), nil
}

func isQueueExistsErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "QueueAlreadyExists") || strings.Contains(s, "already exists")
}

// PublishOrderCreated implements Publisher.
func (c *SQSClient) PublishOrderCreated(ctx context.Context, orderID uuid.UUID) error {
	body, err := MarshalJSONEvent(Event{Type: EventTypeOrderCreated, OrderID: orderID.String()})
	if err != nil {
		return err
	}
	_, err = c.api.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(c.queueURL),
		MessageBody: aws.String(string(body)),
	})
	return err
}

// QueueURL returns the resolved queue URL (for logging/tests).
func (c *SQSClient) QueueURL() string { return c.queueURL }

// ReceiveLoop long-polls SQS and calls handler for each message; deletes on success.
func (c *SQSClient) ReceiveLoop(ctx context.Context, handler func(ctx context.Context, e Event) error) error {
	wait := int32(20)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		out, err := c.api.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     wait,
		})
		if err != nil {
			return err
		}
		for _, m := range out.Messages {
			if m.Body == nil {
				continue
			}
			ev, err := UnmarshalJSONEvent([]byte(*m.Body))
			if err != nil {
				return fmt.Errorf("decode message: %w", err)
			}
			if err := handler(ctx, ev); err != nil {
				return err
			}
			_, err = c.api.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(c.queueURL),
				ReceiptHandle: m.ReceiptHandle,
			})
			if err != nil {
				return err
			}
		}
	}
}

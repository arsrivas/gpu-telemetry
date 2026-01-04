package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"gpu-telemetry/pkg/mq"
)

// MQClient is a lightweight HTTP client used to communicate with
// the custom message queue service.
type MQClient struct {
	BaseURL string
}

// NewMQClient constructs a new MQ client for communicating with
// the message queue service.
func NewMQClient(url string) *MQClient {
	return &MQClient{BaseURL: url}
}

// Poll retrieves envelopes from the queue.
func (c *MQClient) Poll(limit int) ([]mq.Envelope, error) {
	resp, err := http.Get(fmt.Sprintf("%s/poll?limit=%d", c.BaseURL, limit))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("poll failed: %s", resp.Status)
	}

	var envs []mq.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&envs); err != nil {
		return nil, err
	}

	return envs, nil
}

// Enqueue sends an envelope to the queue.
func (c *MQClient) Enqueue(env mq.Envelope) error {
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/enqueue", c.BaseURL),
		"application/json",
		bytes.NewBuffer(b),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("enqueue failed: %s", resp.Status)
	}

	return nil
}

// Ack acknowledges successful processing of a message with the given ID.
func (c *MQClient) Ack(id string) error {
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/ack?id=%s", c.BaseURL, id), bytes.NewBuffer(nil))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

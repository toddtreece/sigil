package tempo

import "context"

type Client struct {
	endpoint string
}

func NewClient(endpoint string) *Client {
	return &Client{endpoint: endpoint}
}

func (c *Client) ForwardTrace(_ context.Context, _ []byte) error {
	// Placeholder: forward OTLP payloads to Tempo in implementation phase.
	return nil
}

func (c *Client) Endpoint() string {
	return c.endpoint
}

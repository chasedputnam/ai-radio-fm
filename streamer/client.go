package streamer

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type IcecastConfig struct {
	Host     string
	Port     int
	Mount    string
	User     string
	Password string
}

type Client struct {
	Config     IcecastConfig
	HTTPClient *http.Client
}

func NewClient(cfg IcecastConfig) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	
	return &Client{
		Config: cfg,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout: 0, // No timeout for streaming
		},
	}
}

func (c *Client) url() string {
	if c.Config.Host == "" {
		return "http://localhost:8000/stream"
	}
	return fmt.Sprintf("http://%s:%d/%s", c.Config.Host, c.Config.Port, c.Config.Mount)
}

// Stream sends the contents of the reader to the Icecast server.
// Reconnects on failure with basic backoff.
func (c *Client) Stream(ctx context.Context, r io.Reader) error {
	// Wait buffer to prevent tight loops
	backoff := 1 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.url(), r)
		if err != nil {
			return err
		}

		req.SetBasicAuth(c.Config.User, c.Config.Password)
		req.Header.Set("Content-Type", "application/ogg")
		req.Header.Set("Ice-Name", "AI Radio FM")
		req.Header.Set("Expect", "100-continue")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		
		// Reset backoff on successful connect
		backoff = 1 * time.Second

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return fmt.Errorf("unauthorized to connect to Icecast")
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			time.Sleep(backoff)
			continue
		}

		// Stream connected, wait for disconnect
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		time.Sleep(backoff)
	}
}

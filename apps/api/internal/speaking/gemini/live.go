package gemini

import (
	"context"
	"github.com/coder/websocket"
	d "github.com/own/mock-ai/apps/api/internal/speaking/domain"
	"net/url"
)

// Credentials never leave the backend; errors deliberately omit the dial URL.
func (c *Client) ConnectLive(ctx context.Context) (*websocket.Conn, error) {
	key, e := c.key(ctx)
	if e != nil {
		return nil, d.ErrUnavailable
	}
	conn, _, e := websocket.Dial(ctx, "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent?key="+url.QueryEscape(key), nil)
	if e != nil {
		return nil, d.ErrUnavailable
	}
	conn.SetReadLimit(2 << 20)
	return conn, nil
}

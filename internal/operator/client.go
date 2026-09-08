package operator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Request struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	ClientID string `json:"client_id,omitempty"`
	Payload  any    `json:"payload,omitempty"`
}

type Response struct {
	ID      string         `json:"id"`
	Success bool           `json:"success"`
	Data    map[string]any `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type StreamChunk struct {
	ID   string         `json:"id"`
	Type string         `json:"type"`
	Data map[string]any `json:"data,omitempty"`
	Done bool           `json:"done,omitempty"`
}

type Client struct {
	conn     net.Conn
	reader   *bufio.Reader
	clientID string

	mu      sync.Mutex
	counter atomic.Int64

	streamMu sync.RWMutex
	streams  map[string]chan StreamChunk

	respMu    sync.RWMutex
	responses map[string]chan Response
}

func NewClient(addr, clientID string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:      conn,
		reader:    bufio.NewReaderSize(conn, 256*1024),
		clientID:  strings.TrimSpace(clientID),
		streams:   make(map[string]chan StreamChunk),
		responses: make(map[string]chan Response),
	}
	go c.readLoop()
	return c, nil
}

func (c *Client) nextID() string {
	return fmt.Sprintf("construct-%d", c.counter.Add(1))
}

func (c *Client) write(req Request) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err = c.conn.Write(append(data, '\n'))
	return err
}

func (c *Client) readLoop() {
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		idBytes, hasID := raw["id"]
		_, hasSuccess := raw["success"]
		_, hasType := raw["type"]
		if !hasID {
			continue
		}

		var id string
		_ = json.Unmarshal(idBytes, &id)

		if hasSuccess {
			var resp Response
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				continue
			}
			c.respMu.RLock()
			ch, ok := c.responses[id]
			c.respMu.RUnlock()
			if ok {
				ch <- resp
			}
			continue
		}

		if hasType {
			var chunk StreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue
			}
			c.streamMu.RLock()
			ch, ok := c.streams[id]
			c.streamMu.RUnlock()
			if ok {
				ch <- chunk
			}
		}
	}
}

func (c *Client) Send(reqType string, payload any) (Response, error) {
	id := c.nextID()
	ch := make(chan Response, 1)

	c.respMu.Lock()
	c.responses[id] = ch
	c.respMu.Unlock()
	defer func() {
		c.respMu.Lock()
		delete(c.responses, id)
		c.respMu.Unlock()
	}()

	if err := c.write(Request{ID: id, Type: reqType, ClientID: c.clientID, Payload: payload}); err != nil {
		return Response{}, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(30 * time.Second):
		return Response{}, fmt.Errorf("timeout waiting for %s", reqType)
	}
}

func (c *Client) Stream(reqType string, payload any) (string, <-chan StreamChunk, error) {
	id := c.nextID()
	ch := make(chan StreamChunk, 256)

	c.streamMu.Lock()
	c.streams[id] = ch
	c.streamMu.Unlock()

	if err := c.write(Request{ID: id, Type: reqType, ClientID: c.clientID, Payload: payload}); err != nil {
		c.streamMu.Lock()
		delete(c.streams, id)
		c.streamMu.Unlock()
		return "", nil, err
	}

	return id, ch, nil
}

func (c *Client) CloseStream(id string) {
	c.streamMu.Lock()
	defer c.streamMu.Unlock()
	if ch, ok := c.streams[id]; ok {
		close(ch)
		delete(c.streams, id)
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

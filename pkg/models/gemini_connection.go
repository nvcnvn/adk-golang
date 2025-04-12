// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// GeminiConnection implements the LlmConnection interface for Gemini models.
type GeminiConnection struct {
	sessionID   string
	stream      io.ReadWriteCloser // Placeholder for bidirectional stream
	sendMutex   sync.Mutex
	receiveChan chan *LlmResponse
	closeChan   chan struct{}
	isClosed    bool
	closeMutex  sync.Mutex
}

// NewGeminiConnection creates a new connection to a Gemini model.
// This is a placeholder implementation as the actual implementation would
// depend on the specific API chosen for bidirectional streaming with Gemini.
func NewGeminiConnection(sessionID string, stream io.ReadWriteCloser) *GeminiConnection {
	conn := &GeminiConnection{
		sessionID:   sessionID,
		stream:      stream,
		receiveChan: make(chan *LlmResponse, 10), // Buffer for received responses
		closeChan:   make(chan struct{}),
	}

	// Start a goroutine to read responses from the stream
	go conn.readResponses()

	return conn
}

// Send sends a message to the model via the connection.
func (c *GeminiConnection) Send(ctx context.Context, content Content) error {
	c.closeMutex.Lock()
	if c.isClosed {
		c.closeMutex.Unlock()
		return errors.New("connection is closed")
	}
	c.closeMutex.Unlock()

	// Ensure only one goroutine can send at a time
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()

	// Convert content to appropriate format for the stream
	// and write it to the stream
	// This is a placeholder implementation
	message := fmt.Sprintf("%v", content)

	// Set a write deadline if context has a deadline
	if deadline, ok := ctx.Deadline(); ok {
		if stream, ok := c.stream.(interface{ SetWriteDeadline(time.Time) error }); ok {
			_ = stream.SetWriteDeadline(deadline)
		}
	}

	// Write the message to the stream
	_, err := io.WriteString(c.stream, message)
	return err
}

// Receive waits for and returns the next response from the model.
func (c *GeminiConnection) Receive(ctx context.Context) (*LlmResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closeChan:
		return nil, errors.New("connection is closed")
	case resp := <-c.receiveChan:
		return resp, nil
	}
}

// Close closes the connection with the model.
func (c *GeminiConnection) Close() error {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	if c.isClosed {
		return nil // Already closed
	}

	c.isClosed = true
	close(c.closeChan)
	return c.stream.Close()
}

// readResponses is an internal method that reads responses from the stream
// and sends them to the receiveChan.
func (c *GeminiConnection) readResponses() {
	defer close(c.receiveChan)

	// This is a placeholder for reading from the stream
	// In a real implementation, this would parse JSON responses from the stream
	buffer := make([]byte, 4096)

	for {
		select {
		case <-c.closeChan:
			return
		default:
			// Read from the stream
			n, err := c.stream.Read(buffer)
			if err != nil {
				if err != io.EOF {
					// Send error response
					c.receiveChan <- &LlmResponse{
						ErrorMessage: fmt.Sprintf("Error reading from stream: %v", err),
					}
				}
				return
			}

			if n > 0 {
				// Parse the response and send it to the receiveChan
				// This is a simplistic placeholder
				text := string(buffer[:n])
				c.receiveChan <- &LlmResponse{
					Content: &Content{
						Parts: []*Part{{
							Text: text,
							Role: "assistant",
						}},
					},
				}
			}
		}
	}
}

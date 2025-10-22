package tests

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/johndosdos/chatter/components/chat"
	"github.com/stretchr/testify/assert"
)

func TestChatInputComponent(t *testing.T) {
	// Create a new buffer to write the component to
	var buf bytes.Buffer

	// Render the component
	err := chat.ChatInput().Render(context.Background(), &buf)

	// Assert that there was no error
	assert.NoError(t, err)

	// Get the rendered HTML
	html := buf.String()

	// Assert that the HTML contains the expected elements
	assert.Contains(t, html, "<form")
	assert.Contains(t, html, `name="content"`)
	assert.Contains(t, html, `type="submit"`)
	assert.Contains(t, html, "Send")
}

func TestChatWindowComponent(t *testing.T) {
	// Dummy userid
	userid := "test-user-123"

	// Create a new buffer to write the component to
	var buf bytes.Buffer

	// Render the component
	err := chat.ChatWindow(userid).Render(context.Background(), &buf)

	// Assert that there was no error
	assert.NoError(t, err)

	// Get the rendered HTML
	html := buf.String()

	// Assert that the HTML contains the expected elements
	assert.Contains(t, html, `ws-connect="/ws?userid=test-user-123"`)
	// Check for ChatInput content
	assert.Contains(t, html, `name="content"`)
	assert.Contains(t, html, "Send")
}

func TestReceiverBubbleComponent(t *testing.T) {
	// With sameUser = false
	var buf bytes.Buffer
	err := chat.ReceiverBubble("test-user", "hello", false, time.Now()).Render(context.Background(), &buf)
	assert.NoError(t, err)
	html := buf.String()
	assert.Contains(t, html, "test-user")
	assert.Contains(t, html, "hello")

	// With sameUser = true
	buf.Reset()
	err = chat.ReceiverBubble("test-user", "world", true, time.Now()).Render(context.Background(), &buf)
	assert.NoError(t, err)
	html = buf.String()
	assert.NotContains(t, html, "test-user")
	assert.Contains(t, html, "world")
}

func TestSenderBubbleComponent(t *testing.T) {
	// With sameUser = false
	var buf bytes.Buffer
	err := chat.SenderBubble("test-user", "hello", false, time.Now()).Render(context.Background(), &buf)
	assert.NoError(t, err)
	html := buf.String()
	assert.Contains(t, html, "test-user")
	assert.Contains(t, html, "hello")

	// With sameUser = true
	buf.Reset()
	err = chat.SenderBubble("test-user", "world", true, time.Now()).Render(context.Background(), &buf)
	assert.NoError(t, err)
	html = buf.String()
	assert.NotContains(t, html, "test-user")
	assert.Contains(t, html, "world")
}

func TestMessageAreaComponent(t *testing.T) {
	var buf bytes.Buffer
	err := chat.MessageArea("test-user-123").Render(context.Background(), &buf)
	assert.NoError(t, err)
	html := buf.String()
	assert.Contains(t, html, `hx-get="/messages?userid=test-user-123"`)
}

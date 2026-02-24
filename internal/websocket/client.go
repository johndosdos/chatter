package websocket

import (
	"context"
	"log"
	"log/slog"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/johndosdos/chatter/components/chat"
	"github.com/johndosdos/chatter/internal/model"
	"golang.org/x/time/rate"
)

type Client struct {
	UserID     uuid.UUID
	Username   string
	conn       *websocket.Conn
	Hub        *Hub
	MessageCh  chan model.ChatMessage
	messageLim *rate.Limiter
	typingLim  *rate.Limiter
	timeWarned time.Time // For rendering the rate limit message. Do not re-render if a message is already there
}

func NewClient(conn *websocket.Conn, userID uuid.UUID, username string) *Client {
	return &Client{
		conn:      conn,
		MessageCh: make(chan model.ChatMessage, 64),
		UserID:    userID,
		Username:  username,
	}
}

func (c *Client) SetMessageLimiter(requests int, window time.Duration) {
	l := rate.NewLimiter(rate.Every(window/time.Duration(requests)), requests)
	c.messageLim = l
}

func (c *Client) SetTypingLimiter(requests int, window time.Duration) {
	l := rate.NewLimiter(rate.Every(window/time.Duration(requests)), requests)
	c.typingLim = l
}

// WriteMessage writes and renders to the outgoing websocket stream.
func (c *Client) WriteMessage(ctx context.Context) {
	// In order to group messages by sender, we need to reference the
	// previous message. We can achieve this by setting the current
	// message as the previous after processing.
	var prevMsg model.ChatMessage
	for {
		select {
		case payload, ok := <-c.MessageCh:
			// We don't want to continue processing when the channel has already been
			// closed.
			if !ok {
				if err := c.conn.Close(websocket.StatusNormalClosure, "channel closed"); err != nil {
					slog.Warn("websocket connection closed", slog.Any("error", err),
						slog.String("reason", websocket.StatusNormalClosure.String()))
				}
				return
			}

			fromSender := payload.UserID == c.UserID
			isSameUserPrevMsg := payload.UserID == prevMsg.UserID

			var content templ.Component
			switch payload.Type {
			case payloadTyping:
				if fromSender {
					continue
				}
				content = chat.TypingIndicator(payload.Username)

			case payloadPresenceCount:
				// We expect a string that contain the count of currently connected users.
				s, err := strconv.Atoi(payload.Content)
				if err != nil {
					log.Printf("failed to convert string to int: %+v", err)
					continue
				}
				content = chat.PresenceCount(s)

			case payloadRateLimit:
				limitWindow := 10 * time.Second // 10s penalty when burst sending 30 messages/min
				timeRemaining := limitWindow - time.Since(c.timeWarned)
				content = chat.RateLimitWarning(int(timeRemaining.Seconds()))

			case payloadMessage:
				if fromSender {
					content = chat.SenderBubble(payload.Username, payload.Content, isSameUserPrevMsg, payload.ID)
				} else {
					content = chat.ReceiverBubble(payload.Username, payload.Content, isSameUserPrevMsg, payload.ID)
				}
			}

			if content == nil {
				continue
			}

			writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			w, err := c.conn.Writer(writeCtx, websocket.MessageText)
			if err != nil {
				slog.WarnContext(ctx, "failed to return a writer",
					"error", err)
				cancel()
				continue
			}

			if err := content.Render(writeCtx, w); err != nil {
				slog.ErrorContext(ctx, "failed to render component",
					"error", err,
					"payload_type", payload.Type,
					"user_id", c.UserID.String(),
					"username", c.Username)
				cancel()
				if err := w.Close(); err != nil {
					slog.Error("writer unexpectedly closed", slog.Any("error", err))
				}
				continue
			}

			if err := w.Close(); err != nil {
				slog.Error("writer unexpectedly closed", slog.Any("error", err))
			}
			cancel()

			// Only update prevMsg for regular messages, not typing indicators.
			if payload.Type == payloadMessage {
				prevMsg = payload
			}

		case <-ctx.Done():
			if err := c.conn.Close(websocket.StatusGoingAway, "context cancelled"); err != nil {
				slog.Warn("websocket connection closed",
					slog.Any("error", err),
					slog.String("reason", websocket.StatusGoingAway.String()))
			}
			return
		}
	}
}

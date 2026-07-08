package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MessageKind int

const (
	//Request
	PlaceBid MessageKind = iota
	// Success
	SuccessfulBid
	// Errors
	FailedToPlaceBid
	InvalidJSON
	// Info
	NewBidPlaced
	AuctionEnded
)

const (
	MAX_MESSAGE_SIZE = 512
	READ_DEADLINE    = 60 * time.Second
	WRITE_AWAIT      = 10 * time.Second
	PING_PERIOD      = (READ_DEADLINE * 9) / 10
)

type Message struct {
	UserId  uuid.UUID   `json:"user_id,omitempty"`
	Message string      `json:"message,omitempty"`
	Amount  int64       `json:"amount,omitempty"`
	Kind    MessageKind `json:"kind"`
}

type AuctionLobby struct {
	sync.Mutex
	Rooms map[uuid.UUID]*AuctionRoom
}

type AuctionRoom struct {
	ID          uuid.UUID
	Context     context.Context
	Broadcast   chan Message
	Register    chan *Client
	Unregister  chan *Client
	Clients     map[uuid.UUID]*Client
	BidsService BidsService
}

func NewAuctionRoom(ctx context.Context, id uuid.UUID, BidsService BidsService) *AuctionRoom {
	return &AuctionRoom{
		ID:          id,
		Context:     ctx,
		Broadcast:   make(chan Message),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		Clients:     make(map[uuid.UUID]*Client),
		BidsService: BidsService,
	}
}

type Client struct {
	Conn   *websocket.Conn
	Room   *AuctionRoom
	Send   chan Message
	UserId uuid.UUID
}

func NewClient(conn *websocket.Conn, room *AuctionRoom, userId uuid.UUID) *Client {
	return &Client{
		Conn:   conn,
		Room:   room,
		Send:   make(chan Message, 512),
		UserId: userId,
	}
}

func (c *Client) ReadEventLoop() {
	defer func() {
		c.Room.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(MAX_MESSAGE_SIZE)
	c.Conn.SetReadDeadline(time.Now().Add(READ_DEADLINE))
	c.Conn.SetPongHandler(func(appData string) error {
		c.Conn.SetReadDeadline(time.Now().Add(READ_DEADLINE))
		return nil
	})

	for {
		var msg Message

		if err := c.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("Unexpected close error", "error", err)
				return
			}
			c.Room.Broadcast <- Message{Kind: InvalidJSON, Message: "invalid json", UserId: c.UserId}
			continue
		}
		msg.UserId = c.UserId
		c.Room.Broadcast <- msg
	}
}

func (c *Client) WriteEventLoop() {
	ticker := time.NewTicker(READ_DEADLINE)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				c.Conn.WriteJSON(Message{Kind: websocket.CloseMessage, Message: "Closing connection"})
				return
			}
			if msg.Kind == AuctionEnded {
				close(c.Send)
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(WRITE_AWAIT))
			err := c.Conn.WriteJSON(msg)
			if err != nil {
				c.Room.Unregister <- c
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(WRITE_AWAIT))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("Unexpected write error", "error", err)
				return
			}
		}
	}
}

func (ar *AuctionRoom) registerClient(c *Client) {
	slog.Info("Registering client", "client_id", c.UserId, "room_id", ar.ID)
	ar.Clients[c.UserId] = c
}

func (ar *AuctionRoom) unregisterClient(c *Client) {
	slog.Info("Unregistering client", "client_id", c.UserId, "room_id", ar.ID)
	delete(ar.Clients, c.UserId)
}

func (ar *AuctionRoom) broadcastMessage(m Message) {
	slog.Info("Broadcasting message", "room_id", ar.ID, "message", m.Message, "userId", m.UserId)

	switch m.Kind {
	case PlaceBid:
		c, ok := ar.Clients[m.UserId]

		if !ok {
			slog.Error("Client not found", "userId", m.UserId)
			return
		}

		_, err := ar.BidsService.PlaceBid(ar.Context, ar.ID, m.UserId, m.Amount)

		if err != nil {
			if errors.Is(err, ErrBidIsTooLow) {
				c.Send <- Message{
					Message: ErrBidIsTooLow.Error(),
					Kind:    FailedToPlaceBid,
				}

			}
			slog.Error("Error placing bid", "error", err)
			return
		}

		c.Send <- Message{
			UserId:  m.UserId,
			Amount:  m.Amount,
			Message: "your bid placed successfully",
			Kind:    SuccessfulBid,
		}

		for id, client := range ar.Clients {
			if id == m.UserId {
				continue
			}
			client.Send <- Message{
				Message: "A new bid has been placed.",
				UserId:  id,
				Amount:  m.Amount,
				Kind:    NewBidPlaced,
			}
		}
	case InvalidJSON:
		c, ok := ar.Clients[m.UserId]

		if !ok {
			slog.Error("Client not found", "userId", m.UserId)
			return
		}

		c.Send <- m
		return
	default:
		slog.Error("Unknown message kind", "kind", m.Kind)
		return
	}

}

func (ar *AuctionRoom) Run() {
	defer func() {
		close(ar.Broadcast)
		close(ar.Register)
		close(ar.Unregister)
	}()

	for {
		select {
		case client := <-ar.Register:
			ar.registerClient(client)
		case client := <-ar.Unregister:
			ar.unregisterClient(client)
		case msg := <-ar.Broadcast:
			ar.broadcastMessage(msg)
		case <-ar.Context.Done():
			slog.Info("Auction room closed")
			for _, client := range ar.Clients {
				client.Send <- Message{
					Message: "Auction has ended",
					Kind:    AuctionEnded,
				}
			}
			return
		}
	}
}

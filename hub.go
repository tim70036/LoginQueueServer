package main

import "github.com/emirpasic/gods/maps/hashmap"

type Hub struct {
	// Registered clients.
	clients *hashmap.Map

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	finishQueue chan string
}

var (
	hub = NewHub()
)

func NewHub() *Hub {
	return &Hub{
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		finishQueue: make(chan string),
		clients:     hashmap.New(),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			logger.Debugf("register client ticketId[%v]", client.ticketId)
			h.clients.Put(client.ticketId, client)
			queue.enter <- client.ticketId
		case client := <-h.unregister:
			logger.Debugf("unregister client ticketId[%v]", client.ticketId)
			if _, doesExist := h.clients.Get(client.ticketId); doesExist {
				h.clients.Remove(client.ticketId)
				queue.leave <- client.ticketId
				close(client.send)
			}
		case ticketId := <-h.finishQueue:
			logger.Debugf("finish queue ticketId[%v]", ticketId)
			if value, doesExist := h.clients.Get(ticketId); doesExist {
				client := value.(*Client)
				client.send <- []byte("finish queue")
				// TODO, close client?
			}
		//The hub handles messages by looping over the registered
		//clients and sending the message to the client's send
		//channel. If the client's send buffer is full, then the
		//hub assumes that the client is dead or stuck. In this
		//case, the hub unregisters the client and closehashashmap
		//websocket.
		case message := <-h.broadcast:
			logger.Debugf("broadcast message[%v]", message)
			for _, value := range h.clients.Values() {
				client := value.(*Client)
				select {
				case client.send <- message:
				default:
					logger.Warnf("id[%v] send channel is full, closing it", client.ticketId)
					h.clients.Remove(client.ticketId)
					queue.leave <- client.ticketId
					close(client.send)
				}
			}
		}
	}
}

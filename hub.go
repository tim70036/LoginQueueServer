package main

type Hub struct {
	// Registered clients.
	clients map[*Client]bool // TODO uid as key?

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

var (
	hub = NewHub()
)

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			logger.Debugf("register id[%v]", client.id)
			h.clients[client] = true
		case client := <-h.unregister:
			logger.Debugf("unregister id[%v]", client.id)
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		//The hub handles messages by looping over the registered
		//clients and sending the message to the client's send
		//channel. If the client's send buffer is full, then the
		//hub assumes that the client is dead or stuck. In this
		//case, the hub unregisters the client and closes the
		//websocket.
		case message := <-h.broadcast:
			logger.Debugf("broadcast message[%v]", message)
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

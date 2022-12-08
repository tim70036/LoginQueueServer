package main

import (
	"encoding/json"
	. "game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"game-soul-technology/joker/joker-login-queue-server/pkg/msg"

	"github.com/emirpasic/gods/maps/hashmap"
)

type ClientRequest struct {
	client    *Client
	wsMessage *msg.WsMessage
}

type Hub struct {
	// Registered clients. Key value: client.ticketId -> client.
	clients *hashmap.Map

	// Stores login request from clients. Key value: client.ticketId -> client.
	loginReqCache *hashmap.Map

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Ws message from clients.
	request chan *ClientRequest

	// Queue finish notification from queue.
	finishQueue chan TicketId
}

var (
	hub = NewHub()
)

func NewHub() *Hub {
	return &Hub{
		clients:       hashmap.New(),
		loginReqCache: hashmap.New(),

		broadcast:   make(chan []byte, 1024),
		register:    make(chan *Client, 1024),
		unregister:  make(chan *Client, 1024),
		request:     make(chan *ClientRequest, 1024),
		finishQueue: make(chan TicketId, 1024),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			Logger.Debugf("register client ticketId[%v]", client.ticketId)
			h.clients.Put(client.ticketId, client)

		case client := <-h.unregister:
			Logger.Debugf("unregister client ticketId[%v]", client.ticketId)

			_, ok := h.clients.Get(client.ticketId)
			if !ok {
				continue
			}

			queue.leave <- client.ticketId
			h.removeClient(client)

		case ticketId := <-h.finishQueue:
			Logger.Debugf("finish queue ticketId[%v]", ticketId)
			value, ok := h.clients.Get(ticketId)
			if !ok {
				Logger.Warnf("finish queue but cannot find client for ticketId[%v]", ticketId)
				continue
			}
			client := value.(*Client)

			value, ok = h.loginReqCache.Get(ticketId)
			if !ok {
				Logger.Warnf("finish queue but cannot find login request info for ticketId[%v]", ticketId)
				continue
			}
			// loginReq := value.(*msg.LoginClientEvent)
			// TODO: login for client

			rawEvent, err := json.Marshal(&msg.LoginServerEvent{
				Jwt: "8787",
			})
			if err != nil {
				Logger.Errorf("cannot marshal LoginServerEvent %v", err)
				continue
			}

			client.sendWsMessage <- &msg.WsMessage{
				EventCode: msg.LoginCode,
				EventData: rawEvent,
			}

			h.removeClient(client)

		case req := <-h.request:
			switch req.wsMessage.EventCode {
			case msg.LoginCode:
				event := &msg.LoginClientEvent{}
				err := json.Unmarshal(req.wsMessage.EventData, event)
				if err != nil {
					Logger.Errorf("ticketId[%v] %v", req.client.ticketId, err)
					continue
				}

				Logger.Debugf("storing event[%+v] into loginReqCache", event)
				h.loginReqCache.Put(req.client.ticketId, event)
				queue.enter <- req.client.ticketId

			default:
				Logger.Errorf("ticketId[%v] invalid eventCode[%v]", req.client.ticketId, req.wsMessage.EventCode)
			}
			//The hub handles messages by looping over the registered
			//clients and sending the message to the client's send
			//channel. If the client's send buffer is full, then the
			//hub assumes that the client is dead or stuck. In this
			//case, the hub unregisters the client and closehashashmap
			//websocket.
			// case message := <-h.broadcast:
			// 	Logger.Debugf("broadcast message[%v]", message)
			// 	for _, value := range h.clients.Values() {
			// 		client := value.(*Client)
			// 		select {
			// 		case client.send <- message:
			// 		default:
			// 			Logger.Warnf("id[%v] send channel is full, closing it", client.ticketId)
			// 			h.clients.Remove(client.ticketId)
			// 			queue.leave <- client.ticketId
			// 			close(client.send)
			// 		}
			// 	}
		}
	}
}

func (h *Hub) removeClient(client *Client) {
	h.clients.Remove(client.ticketId)
	h.loginReqCache.Remove(client.ticketId)
	client.TryClose(false) // Notify client it should close now.
}

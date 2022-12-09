package main

import (
	"encoding/json"
	"fmt"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"game-soul-technology/joker/joker-login-queue-server/pkg/msg"
	"os"

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
	loginDataCache *hashmap.Map

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Ws message from clients.
	request chan *ClientRequest

	queue *Queue
}

func ProvideHub(queue *Queue) *Hub {
	return &Hub{
		clients:        hashmap.New(),
		loginDataCache: hashmap.New(),

		broadcast:  make(chan []byte, 1024),
		register:   make(chan *Client, 1024),
		unregister: make(chan *Client, 1024),
		request:    make(chan *ClientRequest, 1024),

		queue: queue,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			logger.Debugf("register client ticketId[%v]", client.ticketId)
			h.clients.Put(client.ticketId, client)

		case client := <-h.unregister:
			logger.Debugf("unregister client ticketId[%v]", client.ticketId)

			_, ok := h.clients.Get(client.ticketId)
			if !ok {
				continue
			}

			h.queue.leave <- client.ticketId
			h.removeClient(client)

		case ticketId := <-h.queue.finish:
			logger.Debugf("finish queue ticketId[%v]", ticketId)
			value, ok := h.clients.Get(ticketId)
			if !ok {
				logger.Warnf("finish queue but cannot find client for ticketId[%v]", ticketId)
				continue
			}
			client := value.(*Client)

			value, ok = h.loginDataCache.Get(ticketId)
			if !ok {
				logger.Warnf("finish queue but cannot find login request info for ticketId[%v]", ticketId)
				continue
			}
			loginData := value.(*msg.LoginClientEvent)

			authResult := make(chan string)
			go h.loginForClient(loginData, authResult)
			go h.sendClientToLogin(client, authResult)

		case req := <-h.request:
			switch req.wsMessage.EventCode {
			case msg.LoginCode:
				event := &msg.LoginClientEvent{}
				err := json.Unmarshal(req.wsMessage.EventData, event)
				if err != nil {
					logger.Errorf("ticketId[%v] %v", req.client.ticketId, err)
					continue
				}

				logger.Debugf("storing event[%+v] into loginReqCache", event)
				h.loginDataCache.Put(req.client.ticketId, event)
				h.queue.enter <- req.client.ticketId

			default:
				logger.Errorf("ticketId[%v] invalid eventCode[%v]", req.client.ticketId, req.wsMessage.EventCode)
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
	// TOOD: add lock
	h.clients.Remove(client.ticketId)
	h.loginDataCache.Remove(client.ticketId)
	client.TryClose(false) // Notify client it should close now.
}

func (h *Hub) loginForClient(loginData *msg.LoginClientEvent, result chan<- string) {
	defer close(result)

	var (
		url     string = os.Getenv("MAIN_SERVER_HOST") + "/api/user/authorization"
		payload string
	)
	switch loginData.Type {
	case "apple":
		url += "/apple"
		payload = fmt.Sprintf(`{"accessToken":"%v"}`, loginData.Token)
	case "device":
		url += "/device"
		payload = fmt.Sprintf(`{"uniqueId":"%v"}`, loginData.Token)
	case "facebook":
		url += "/facebook"
		payload = fmt.Sprintf(`{"token":"%v"}`, loginData.Token)
	case "google":
		url += "/google"
		payload = fmt.Sprintf(`{"token":"%v"}`, loginData.Token)
	case "line":
		url += "/line"
		payload = fmt.Sprintf(`{"accessToken":"%v"}`, loginData.Token)
	default:
		logger.Errorf("invalid login type[%v]", loginData.Type)
		return
	}

	authData := &struct {
		Data struct {
			Jwt string `json:"jwt"`
		} `json:"data"`
	}{}

	resp, err := infra.HttpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("platform", "Android"). // TODO
		SetBody(payload).
		SetResult(authData).
		Post(url)

	if err != nil {
		logger.Errorf("request failed %v", err)
		return
	}

	if resp.IsError() {
		logger.Errorf("request failed with status[%v]", resp.Status)
		return
	}

	result <- authData.Data.Jwt
}

func (h *Hub) sendClientToLogin(client *Client, result <-chan string) {
	defer h.removeClient(client)

	jwt, ok := <-result
	if !ok {
		logger.Warnf("cannot get login credential from closed channel")
		return
	}

	rawEvent, err := json.Marshal(&msg.LoginServerEvent{
		Jwt: jwt,
	})
	if err != nil {
		logger.Errorf("cannot marshal LoginServerEvent %v", err)
		return
	}

	client.sendWsMessage <- &msg.WsMessage{
		EventCode: msg.LoginCode,
		EventData: rawEvent,
	}
}

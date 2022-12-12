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
	wsRequest chan *ClientRequest

	queue *Queue
}

func ProvideHub(queue *Queue) *Hub {
	return &Hub{
		clients:        hashmap.New(),
		loginDataCache: hashmap.New(),

		broadcast:  make(chan []byte, 1024),
		register:   make(chan *Client, 1024),
		unregister: make(chan *Client, 1024),
		wsRequest:  make(chan *ClientRequest, 1024),

		queue: queue,
	}
}

func (h *Hub) Run() {
	go h.handleClient()
	go h.handleQueue()
}

func (h *Hub) handleClient() {
	for {
		select {
		case client := <-h.register:
			logger.Debugf("register client ticketId[%v] ip[%v]", client.ticketId, client.ip)
			h.clients.Put(client.ticketId, client)

		case client := <-h.unregister:
			logger.Debugf("unregister client ticketId[%v]", client.ticketId)

			_, ok := h.clients.Get(client.ticketId)
			if !ok {
				continue
			}

			h.queue.leave <- client.ticketId
			h.removeClient(client)

		case req := <-h.wsRequest:
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
		}
	}
}

func (h *Hub) handleQueue() {
	for {
		select {
		case ticket := <-h.queue.notifyDirtyTicket:
			logger.Debugf("notifyDirtyTicket ticketId[%v]", ticket.ticketId)
			value, ok := h.clients.Get(ticket.ticketId)
			if !ok {
				logger.Warnf("notifyDirtyTicket but cannot find client for ticketId[%v]", ticket.ticketId)
				continue
			}

			rawEvent, err := json.Marshal(&msg.TicketServerEvent{
				TicketId: string(ticket.ticketId),
				Position: ticket.position,
			})
			if err != nil {
				logger.Errorf("cannot marshal TicketServerEvent %v", err)
				return
			}

			wsMessage := &msg.WsMessage{
				EventCode: msg.TicketCode,
				EventData: rawEvent,
			}

			client := value.(*Client)
			client.sendWsMessage <- wsMessage

		case stats := <-h.queue.notifyStats:
			logger.Debugf("notifyStats stats[%+v]", stats)
			rawEvent, err := json.Marshal(&msg.QueueStatsServerEvent{
				ActiveTickets: stats.activeTickets,
				HeadPosition:  stats.headPosition,
				TailPosition:  stats.tailPosition,
				AvgWaitMsec:   stats.avgWaitDuration.Milliseconds(),
			})
			if err != nil {
				logger.Errorf("cannot marshal QueueStatsServerEvent %v", err)
				return
			}

			wsMessage := &msg.WsMessage{
				EventCode: msg.QueueStatsCode,
				EventData: rawEvent,
			}

			for _, value := range h.clients.Values() {
				client := value.(*Client)
				client.sendWsMessage <- wsMessage
			}

		case ticketId := <-h.queue.notifyFinish:
			logger.Debugf("notifyFinish ticketId[%v]", ticketId)
			value, ok := h.clients.Get(ticketId)
			if !ok {
				logger.Warnf("notifyFinish but cannot find client for ticketId[%v]", ticketId)
				continue
			}
			client := value.(*Client)

			value, ok = h.loginDataCache.Get(ticketId)
			if !ok {
				logger.Warnf("notifyFinish but cannot find login request info for ticketId[%v]", ticketId)
				continue
			}
			loginData := value.(*msg.LoginClientEvent)

			authResult := make(chan string)
			go h.loginForClient(loginData, client, authResult)
			go h.finishClient(client, authResult)
		}
	}
}

func (h *Hub) removeClient(client *Client) {
	// TOOD: add lock
	h.clients.Remove(client.ticketId)
	h.loginDataCache.Remove(client.ticketId)
	client.TryClose(false) // Notify client it should close now.
}

func (h *Hub) loginForClient(loginData *msg.LoginClientEvent, client *Client, result chan<- string) {
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

	// TODO hwo to send client IP
	resp, err := infra.HttpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("platform", client.platform).
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

	logger.Infof("login success for ticketId[%v]", client.ticketId)
	result <- authData.Data.Jwt
}

func (h *Hub) finishClient(client *Client, result <-chan string) {
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

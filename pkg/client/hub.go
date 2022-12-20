package client

import (
	"encoding/json"
	"fmt"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"game-soul-technology/joker/joker-login-queue-server/pkg/msg"
	"game-soul-technology/joker/joker-login-queue-server/pkg/queue"
	"os"

	"github.com/emirpasic/gods/maps/hashmap"
	"github.com/imroc/req/v3"
	"go.uber.org/zap"
)

type ClientRequest struct {
	client    *Client
	wsMessage *msg.WsMessage
}

type Hub struct {
	// Registered clients. Key value: client.id -> client.
	clients *hashmap.Map

	// Stores login request from clients. Key value: client.id -> client.
	loginDataCache *hashmap.Map

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Ws message from clients.
	wsRequest chan *ClientRequest

	queue *queue.Queue

	httpClient *req.Client

	logger *zap.SugaredLogger
}

func ProvideHub(queue *queue.Queue, httpClient *req.Client, loggerFactory *infra.LoggerFactory) *Hub {
	return &Hub{
		clients:        hashmap.New(),
		loginDataCache: hashmap.New(),

		broadcast:  make(chan []byte, 1024),
		register:   make(chan *Client, 1024),
		unregister: make(chan *Client, 1024),
		wsRequest:  make(chan *ClientRequest, 1024),

		queue:      queue,
		httpClient: httpClient,
		logger:     loggerFactory.Create("Hub").Sugar(),
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
			h.logger.Debugf("register client id[%v] ip[%v]", client.id, client.ip)
			h.clients.Put(client.id, client)

			rawEvent, err := json.Marshal(&msg.ShouldQueueEvent{
				ShouldQueue: true,
			})
			if err != nil {
				h.logger.Errorf("cannot marshal ShouldQueueEvent %v", err)
				return
			}

			wsMessage := &msg.WsMessage{
				EventCode: msg.ShouldQueueCode,
				EventData: rawEvent,
			}
			client.sendWsMessage <- wsMessage

		case client := <-h.unregister:
			h.logger.Debugf("unregister client id[%v]", client.id)

			_, ok := h.clients.Get(client.id)
			if !ok {
				continue
			}

			h.queue.Leave <- queue.TicketId(client.id)
			h.removeClient(client)

		case req := <-h.wsRequest:
			switch req.wsMessage.EventCode {
			case msg.LoginCode:
				event := &msg.LoginClientEvent{}
				err := json.Unmarshal(req.wsMessage.EventData, event)
				if err != nil {
					h.logger.Errorf("id[%v] %v", req.client.id, err)
					continue
				}

				h.logger.Debugf("storing event[%+v] into loginReqCache", event)
				h.loginDataCache.Put(req.client.id, event)
				h.queue.Enter <- queue.TicketId(req.client.id)

			default:
				h.logger.Errorf("id[%v] invalid eventCode[%v]", req.client.id, req.wsMessage.EventCode)
			}
		}
	}
}

func (h *Hub) handleQueue() {
	for {
		select {
		case ticket := <-h.queue.NotifyTicket:
			h.logger.Debugf("notifyDirtyTicket ticketId[%v]", ticket.TicketId)
			value, ok := h.clients.Get(string(ticket.TicketId))
			if !ok {
				h.logger.Warnf("notifyDirtyTicket but cannot find client for ticketId[%v]", ticket.TicketId)
				continue
			}

			rawEvent, err := json.Marshal(&msg.TicketServerEvent{
				TicketId: string(ticket.TicketId),
				Position: ticket.Position,
			})
			if err != nil {
				h.logger.Errorf("cannot marshal TicketServerEvent %v", err)
				return
			}

			wsMessage := &msg.WsMessage{
				EventCode: msg.TicketCode,
				EventData: rawEvent,
			}

			client := value.(*Client)
			client.sendWsMessage <- wsMessage

		case stats := <-h.queue.NotifyStats:
			h.logger.Debugf("notifyStats stats[%+v]", stats)
			rawEvent, err := json.Marshal(&msg.QueueStatsServerEvent{
				HeadPosition: stats.HeadPosition,
				TailPosition: stats.TailPosition,
				AvgWaitMsec:  stats.AvgWaitDuration.Milliseconds(),
			})
			if err != nil {
				h.logger.Errorf("cannot marshal QueueStatsServerEvent %v", err)
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

		case ticketId := <-h.queue.NotifyFinish:
			h.logger.Debugf("notifyFinish ticketId[%v]", ticketId)
			value, ok := h.clients.Get(string(ticketId))
			if !ok {
				h.logger.Warnf("notifyFinish but cannot find client for ticketId[%v]", ticketId)
				continue
			}
			client := value.(*Client)

			value, ok = h.loginDataCache.Get(string(ticketId))
			if !ok {
				h.logger.Warnf("notifyFinish but cannot find login request info for ticketId[%v]", ticketId)
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
	// TODO: add lock
	h.clients.Remove(client.id)
	h.loginDataCache.Remove(client.id)
	client.TryClose(false) // Notify client it should close now.
}

func (h *Hub) loginForClient(loginData *msg.LoginClientEvent, client *Client, result chan<- string) {
	defer close(result)

	var (
		url     string = os.Getenv("MAIN_SERVER_HOST") + "/api/user/authorization"
		payload string
	)
	switch loginData.Type {
	case msg.AppleLogin:
		url += "/apple"
		payload = fmt.Sprintf(`{"accessToken":"%v"}`, loginData.Token)
	case msg.DeviceLogin:
		url += "/device"
		payload = fmt.Sprintf(`{"uniqueId":"%v"}`, loginData.Token)
	case msg.FacebookLogin:
		url += "/facebook"
		payload = fmt.Sprintf(`{"token":"%v"}`, loginData.Token)
	case msg.GoogleLogin:
		url += "/google"
		payload = fmt.Sprintf(`{"token":"%v"}`, loginData.Token)
	case msg.LineLogin:
		url += "/line"
		payload = fmt.Sprintf(`{"accessToken":"%v"}`, loginData.Token)
	default:
		h.logger.Errorf("invalid login type[%v]", loginData.Type)
		return
	}

	authData := &struct {
		Data struct {
			Jwt string `json:"jwt"`
		} `json:"data"`
	}{}

	// TODO hwo to send client IP
	resp, err := h.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("platform", client.platform).
		SetBody(payload).
		SetResult(authData).
		Post(url)

	if err != nil {
		h.logger.Errorf("request failed %v", err)
		return
	}

	if resp.IsError() {
		h.logger.Errorf("request failed with status[%v]", resp.Status)
		return
	}

	h.logger.Infof("login success for id[%v]", client.id)
	result <- authData.Data.Jwt
}

func (h *Hub) finishClient(client *Client, result <-chan string) {
	defer h.removeClient(client)

	jwt, ok := <-result
	if !ok {
		h.logger.Warnf("cannot get login credential from closed channel")
		return
	}

	rawEvent, err := json.Marshal(&msg.LoginServerEvent{
		Jwt: jwt,
	})
	if err != nil {
		h.logger.Errorf("cannot marshal LoginServerEvent %v", err)
		return
	}

	client.sendWsMessage <- &msg.WsMessage{
		EventCode: msg.LoginCode,
		EventData: rawEvent,
	}
}

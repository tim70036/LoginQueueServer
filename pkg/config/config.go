package config

import "flag"

type Config struct {
	SessionStaleSeconds *int
	TicketStaleSeconds  *int

	NotifyStatsIntervalSeconds *int
	DequeueIntervalSeconds     *int
	MaxDequeuePerInterval      *int

	InitAvgWaitSeconds    *int
	AverageWaitWindowSize *int

	PingIntervalSeconds *int
}

var CFG = &Config{
	SessionStaleSeconds:        flag.Int("session-stale-seconds", 300, "The number of seconds before a session is considered stale. If client goes offline over this period of time, he has to go into login queue again."),
	TicketStaleSeconds:         flag.Int("ticket-stale-seconds", 300, "After client is inactive for this period, ticket is viewed as stale and can be removed (not immediately removed). If client come back, he will have to wait from the start of the queue."),
	NotifyStatsIntervalSeconds: flag.Int("notify-stats-interval-seconds", 5, "Interval to notify stats to client."),
	DequeueIntervalSeconds:     flag.Int("dequeue-interval-seconds", 10, "Interval to dequeue tickets."),
	MaxDequeuePerInterval:      flag.Int("max-dequeue-per-interval", 500, "Max number of tickets to dequeue per interval."),
	InitAvgWaitSeconds:         flag.Int("init-avg-wait-seconds", 180, "Initial default value of wait duration."),
	AverageWaitWindowSize:      flag.Int("average-wait-window-size", 50, "The size of sliding window for calculating average wait time of a ticket."),
	PingIntervalSeconds:        flag.Int("ping-interval-seconds", 30, "Send pings to websocket peer with this interval."),
}

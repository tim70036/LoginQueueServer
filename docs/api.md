# Connection Establishment

Utilize websocket client and connect to `wss://{LOGIN_QUEUE_SERVER_HOST}:{SERVER_PORT}`.

You will need to place `platform` and `id` in the request header for the connection. `id` is a client generated string. It’s used as identifier across multiple frontend sessions. Should be unique enough (eg. guid). `platform` is the required payload for main server login, see main server document for futher feature.

 

# Heartbeat

In order to detect unexpected disconnection, server will periodically send ping to client. Client must send pong back to server to maintain the connection. Client can try reconnect if it doesn’t receive server ping for a while.



# Websocket Message Format

All messages are in the format of JSON string. Each message contains - eventCode that defines what type of event this message contains. The data of this event is stored in eventData.
```
{
  "- eventCode": number,
  "eventData": EVENT_DATA (will always be a JSON object)
}
```

# Websocket Event

This section defines the data format for each type of event. The data will be located in `eventData` field of websocket message.

## ShouldQueue

- eventCode 1000
- ServerWsEvent. If received true, then client is good to go. Do not wait for further message.
```
{
  shouldQueue: true
}
```

## Login

- eventCode 1001
- ClientWsEvent (Request insert a ticket into queue with login info)
```
{
  "type": 0
  "token": "asdz23asda-123sac"
}
```

- ServerWsEvent (Response after login with jwt session credential)
```
{
  "jwt": "iamjwtjwtjwtjwtjwtjwtjwt"
}
```

- The login type in ClientWsEvent:
```
const (
	FacebookLogin = 0
	GoogleLogin   = 1
	AppleLogin    = 2
	LineLogin     = 3
	DeviceLogin   = 4
)
```


## QueueStats

- eventCode 1002
- ServerWsEvent
```
{
  "headPosition": 1,
  "tailPosition": 5,
  "avgWaitMsec": 17000 // For a ticket
}
```

## Ticket

- eventCode 1003
- ServerWsEvent
```
{
  "ticketId": "12adccasxax",
  "position": 87
}
```


# Position

From QueueStats and Ticket event, client will have three position data and deduct the following information:

- How many tickets is in front of this client = `Ticket.position` - `QueueStats.headPosition`
- How many tickets is in back of this client = `QueueStats.tailPosition` - `Ticket.position`

# joker-login-queue-server
A queue server written in Golang that queue client login requests to
avoid main server outrage. If user traffic surges, we need to protect
our main server. Queue server will read the user number that are
currently in main server and decide how much new user can login to
main server by dequeueing certain number of login requests out from
its queue. It uses websocket protocol to communicate with frontend
client and request additional data from main server. However, since
queue server stores client info in server memory, it's not possible to
scale queue server to multiple machines. This is left for future
peformance optimization (Not that easy).

## Prerequisites

First, ensure that you are using a machine meeting the following requirements:

- Have `docker` and `docker-compose` installed.
- Have a keyboard for you to type command.
- Chill enough.

## Installation
1. Set credential in the file `.env` and place it in project root
   directory. This file is read by docker compose:
    ```
        // Port of the queue server.
        SERVER_PORT="5487" 

        // Queue server need a redis instance to store data.
        REDIS_HOST="host.docker.internal:8787" 
        REDIS_DB="0"

        // The main server host and required api key to make request.
        MAIN_SERVER_HOST="http://host.docker.internal:8888" 
        MAIN_SERVER_API_KEY="d7153da6-aa6f-4a7b-9c30-a9fc92708bae"

        // Queue server tls certificate
        TLS_PRIVATE_KEY_PATH="deploy/certs/game-soul-swe.com/private.key" 
        TLS_CERT_PATH="deploy/certs/game-soul-swe.com/public.crt"

    ```
2. Put in TLS certificate in `deploy/certs` directory. Remember to match the path you fill for `TLS_PRIVATE_KEY_PATH` and `TLS_CERT_PATH` for `.env`

Finally, run `docker-compose build` and `docker-compose up -d` to run the tool.

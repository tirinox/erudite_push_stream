# Erudite push stream

It's like nginx push stream module.

Works via *TCP* sockets, not WebSockets.

Includes Docker's image.

### Configuration

The app is configured through environment variables:

`ENABLE_LOG=1` - verbose logging mode.

`BIND=12345` - to change the port that the app is listening to. Default port is 10026.

`API_KEY=you_api_key` - for protecting the app from unauthorized message publishers (default is an empty string).

### Commands

You can connect to the app running locally using the following command:
`telnet localhost 10026`

Then clients must register providing their ID strings:

`{"command":"register","ident":"123"}`

Otherwise unregisted client will be disconnected soon (20 sec).
Moreover clients must respond command "pong" or they will be disconnected too after the next ping:

`{"command":"pong"}`

Server publishes a message to clients with "123" ID:

`{"command":"publish","api_key":"","message":"Test message","ident":"123"}`

The app returns a message and closes the connection:

`{"ident":"123","result":"false","sv_time":1476176086}`

*result* means whether the message was delivered or not.

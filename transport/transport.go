package transport

import (
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/subscription"
	"time"
)

//Options represents the connection options to be used by a transport
type Options struct {
	Url           string
	RetryInterval time.Duration

	InExt  []message.Extension
	OutExt []message.Extension
	//todo dial timeout
	//todo read/write deadline
}

//Transport represents the transport to be used to comunicate with the faye server
type Transport interface {
	//Name returns the transport name
	Name() string
	//Init initializes the transport with the provided options
	Init(options *Options) error
	//Options return the transport Options
	Options() *Options
	//Handshake initiates a connection negotiation by sending a message to the /meta/handshake channel.
	Handshake() error
	//Connect is called  after a client has discovered the server’s capabilities with a handshake exchange,
	//a connection is established by sending a message to the /meta/connect channel
	Connect() error
	//Disconnect closes all subscriptions and inform the server to remove any client-related state.
	//any subsequent method call to the transport object will result in undefined behaviour.
	Disconnect() error
	//Subscribe informs the server that messages published to that channel are delivered to itself.
	Subscribe(channel string) (*subscription.Subscription, error)
	//Unsubscribe informs the server that the client will no longer listen to incoming event messages on
	//the specified channel/subscription
	Unsubscribe(sub *subscription.Subscription) error
	//Publish publishes events on a channel by sending event messages, the server MAY  respond to a publish event
	//if this feature is supported by the server use the OnPublishResponse to get the publish status.
	Publish(subscription string, message message.Data) (id string, err error)
	//OnPublishResponse sets the handler to be triggered if the server replies to the publish request
	//according to the spec the server MAY reply to the publish request, so its not guaranteed that this handler will
	//ever be triggered
	//can be used to identify the status of the published request and for example retry failed published requests
	OnPublishResponse(subscription string, onMsg func(message *message.Message))
}

//MetaMessage are channels commencing with the /meta/ segment ans, are the channels used by the faye protocol itself.
type MetaMessage = string

const (
	MetaSubscribe   MetaMessage = "/meta/subscribe"
	MetaConnect     MetaMessage = "/meta/connect"
	MetaDisconnect  MetaMessage = "/meta/disconnect"
	MetaUnsubscribe MetaMessage = "/meta/unsubscribe"
	MetaHandshake   MetaMessage = "/meta/handshake"
)

//EventMessage are published in event messages sent from a faye client to a faye server
//and are delivered in event messages sent from a faye server to a faye client.
type EventMessage = int

const (
	//
	EventPublish EventMessage = iota
	EventDelivery
)

type Reconnect = string

const (
	//ReconnectRetry indicates that a client MAY attempt to reconnect with a /meta/connect message,
	//after the interval (as defined by interval advice field or client-default backoff), and with the same credentials.
	ReconnectRetry Reconnect = "retry"

	//ReconnectHandshake indicates that the server has terminated any prior connection status and the client MUST reconnect
	// with a /meta/handshake message.
	//A client MUST NOT automatically retry when a reconnect advice handshake has been received.
	ReconnectHandshake Reconnect = "handshake"

	//ReconnectNone indicates a hard failure for the connect attempt.
	//A client MUST respect reconnect advice none and MUST NOT automatically retry or handshake.
	ReconnectNone Reconnect = "none"
)

var metaMessages = []MetaMessage{MetaSubscribe, MetaConnect, MetaUnsubscribe, MetaHandshake, MetaDisconnect}

func IsMetaMessage(msg *message.Message) bool {
	for i := range metaMessages {
		if msg.Channel == metaMessages[i] {
			return true
		}
	}
	return false
}

func IsEventDelivery(msg *message.Message) bool {
	if msg.Data != nil {
		return true
	}
	return false
}

func IsEventPublish(msg *message.Message) bool {
	return !IsEventDelivery(msg)
}

var registeredTransports = map[string]Transport{}

func RegisterTransport(t Transport) {
	registeredTransports[t.Name()] = t //todo validate
}

func GetTransport(name string) Transport {
	return registeredTransports[name]
}

package fayec

import (
	"github.com/thesyncim/faye/internal/dispatcher"
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/subscription"
	"github.com/thesyncim/faye/transport"
	_ "github.com/thesyncim/faye/transport/websocket"
)

type options struct {
	transport     transport.Transport
	transportOpts transport.Options
	extensions    message.Extensions
}

var defaultOpts = options{
	transport: transport.GetTransport("websocket"),
}

//https://faye.jcoglan.com/architecture.html
type client interface {
	Disconnect() error
	Subscribe(subscription string) (*subscription.Subscription, error)
	Publish(subscription string, message message.Data) error

	//SetOnTransportDownHandler(onTransportDown func(err error))
	//SetOnTransportUpHandler(onTransportUp func())
}

//Option set the Client options, such as Transport, message extensions,etc.
type Option func(*options)

var _ client = (*Client)(nil)

// Client represents a client connection to an faye server.
type Client struct {
	opts       options
	dispatcher *dispatcher.Dispatcher
}

//NewClient creates a new faye client with the provided options and connect to the specified url.
func NewClient(url string, opts ...Option) (*Client, error) {
	var c Client
	c.opts = defaultOpts
	for _, opt := range opts {
		opt(&c.opts)
	}

	c.dispatcher = dispatcher.NewDispatcher(url, c.opts.transportOpts, c.opts.extensions)
	c.dispatcher.SetTransport(c.opts.transport)
	err := c.dispatcher.Connect()
	if err != nil {
		return nil, err
	}

	return &c, nil
}

//Subscribe informs the server that messages published to that channel are delivered to itself.
func (c *Client) Subscribe(subscription string) (*subscription.Subscription, error) {
	return c.dispatcher.Subscribe(subscription)
}

//Publish publishes events on a channel by sending event messages, the server MAY  respond to a publish event
//if this feature is supported by the server use the OnPublishResponse to get the publish status.
func (c *Client) Publish(subscription string, data message.Data) (err error) {
	return c.dispatcher.Publish(subscription, data)
}

//Disconnect closes all subscriptions and inform the server to remove any client-related state.
//any subsequent method call to the client object will result in undefined behaviour.
func (c *Client) Disconnect() error {
	return c.dispatcher.Disconnect()
}

//WithOutExtension append the provided outgoing extension to the the default transport options
//extensions run in the order that they are provided
func WithOutExtension(extension message.Extension) Option {
	return func(o *options) {
		o.extensions.Out = append(o.extensions.Out, extension)
	}
}

//WithExtension append the provided incoming extension and outgoing to the list of incoming and outgoing extensions.
//extensions run in the order that they are provided
func WithExtension(inExt message.Extension, outExt message.Extension) Option {
	return func(o *options) {
		o.extensions.In = append(o.extensions.In, inExt)
		o.extensions.Out = append(o.extensions.Out, outExt)
	}
}

//WithInExtension append the provided incoming extension to the list of incoming extensions.
//extensions run in the order that they are provided
func WithInExtension(extension message.Extension) Option {
	return func(o *options) {
		o.extensions.In = append(o.extensions.In, extension)
	}
}

//WithTransport sets the client transport to be used to communicate with server.
func WithTransport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}

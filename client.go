package fayec

import (
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/transport"
	_ "github.com/thesyncim/faye/transport/websocket"
)

type options struct {
	inExt     []message.Extension
	outExt    []message.Extension
	transport transport.Transport
}

var defaultOpts = options{
	transport: transport.GetTransport("websocket"),
}

//https://faye.jcoglan.com/architecture.html
type client interface {
	Disconnect() error
	Subscribe(subscription string, onMessage func(message message.Data)) error
	Unsubscribe(subscription string) error
	Publish(subscription string, message message.Data) (string, error)
	OnPublishResponse(subscription string, onMsg func(message *message.Message))
}

type Option func(*options)

var _ client = (*Client)(nil)

type Client struct {
	opts options
}

func NewClient(url string, opts ...Option) (*Client, error) {
	var c Client
	c.opts = defaultOpts
	for _, opt := range opts {
		opt(&c.opts)
	}
	tops := &transport.Options{
		Url:    url,
		InExt:  c.opts.inExt,
		OutExt: c.opts.outExt,
	}
	var err error
	if err = c.opts.transport.Init(tops); err != nil {
		return nil, err
	}

	if err = c.opts.transport.Handshake(); err != nil {
		return nil, err
	}

	if err = c.opts.transport.Connect(); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *Client) Subscribe(subscription string, onMsg func(message message.Data)) error {
	return c.opts.transport.Subscribe(subscription, onMsg)
}

func (c *Client) Unsubscribe(subscription string) error {
	return c.opts.transport.Unsubscribe(subscription)
}

func (c *Client) Publish(subscription string, data message.Data) (id string, err error) {
	return c.opts.transport.Publish(subscription, data)
}

//OnPublishResponse sets the handler to be triggered if the server replies to the publish request
//according to the spec the server MAY reply to the publish request, so its not guaranteed that this handler will
//ever be triggered
//can be used to identify the status of the published request and for example retry failed published requests
func (c *Client) OnPublishResponse(subscription string, onMsg func(message *message.Message)) {
	c.opts.transport.OnPublishResponse(subscription, onMsg)
}

func (c *Client) Disconnect() error {
	return c.opts.transport.Disconnect()
}

func WithOutExtension(extension message.Extension) Option {
	return func(o *options) {
		o.outExt = append(o.outExt, extension)
	}
}

func WithExtension(inExt message.Extension, outExt message.Extension) Option {
	return func(o *options) {
		o.inExt = append(o.inExt, inExt)
		o.outExt = append(o.outExt, outExt)
	}
}

func WithInExtension(extension message.Extension) Option {
	return func(o *options) {
		o.inExt = append(o.inExt, extension)
	}
}

func WithTransport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}

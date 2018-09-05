package fayec

import (
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/transport"
	_ "github.com/thesyncim/faye/transport/websocket"
)

type options struct {
	inExt     message.Extension
	outExt    message.Extension
	transport transport.Transport
}

var defaultOpts = options{
	transport: transport.GetTransport("websocket"),
}

//https://faye.jcoglan.com/architecture.html
type client interface {
	Subscribe(subscription string, onMsg func(message *message.Message)) error
	Publish(subscription string, message *message.Message) error
	//todo unsubscribe,etc
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

func WithOutExtension(extension message.Extension) Option {
	return func(o *options) {
		o.outExt = extension
	}
}

func WithInExtension(extension message.Extension) Option {
	return func(o *options) {
		o.inExt = extension
	}
}

func WithTransport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}

func (c *Client) Subscribe(subscription string, onMsg func(message *message.Message)) error {
	return c.opts.transport.Subscribe(subscription, onMsg)
}

func (c *Client) Publish(subscription string, message *message.Message) error {
	return c.opts.transport.Publish(subscription, message)
}

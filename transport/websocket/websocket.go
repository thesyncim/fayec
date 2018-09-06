package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/transport"

	"strconv"
	"sync"
	"sync/atomic"
)

const transportName = "websocket"

func init() {
	transport.RegisterTransport(&Websocket{})
}

type Websocket struct {
	TransportOpts *transport.Options
	conn          *websocket.Conn
	clientID      string
	msgID         *uint64
	once          sync.Once

	subsMu sync.Mutex //todo sync.Map
	subs   map[string]chan *message.Message
}

var _ transport.Transport = (*Websocket)(nil)

func (w *Websocket) Init(options *transport.Options) error {
	var (
		err   error
		msgID uint64
	)
	w.TransportOpts = options
	w.msgID = &msgID
	w.subs = map[string]chan *message.Message{}
	w.conn, _, err = websocket.DefaultDialer.Dial(options.Url, nil)
	if err != nil {
		return err
	}
	return nil
}

func (w *Websocket) readWorker() error {
	for {
		var payload []message.Message
		err := w.conn.ReadJSON(&payload)
		if err != nil {
			return err
		}
		//dispatch
		msg := &payload[0]

		if transport.IsControlMsg(msg.Channel) {
			//handle it
			switch msg.Channel {
			case transport.Subscribe:
				//handle Subscribe resp
				if !msg.Successful {
					w.subsMu.Lock()
					subscription, ok := w.subs[msg.Subscription]
					w.subsMu.Unlock()
					if !ok {
						panic("BUG: subscription not registered `" + msg.Subscription + "`")
					}
					if msg.GetError() != nil {
						//inject the error
						msg.Error = fmt.Sprintf("susbscription `%s` failed", msg.Subscription)
					}
					subscription <- msg
					close(subscription)
					w.subsMu.Lock()
					delete(w.subs, msg.Channel)
					w.subsMu.Unlock()
				}
			case transport.Unsubscribe:
				//handle Unsubscribe resp
			case transport.Connect:
				//handle Connect resp

			case transport.Disconnect:
				//handle Disconnect resp

			case transport.Handshake:
				//handle Handshake resp
			}

			continue
		}

		w.subsMu.Lock()
		subscription := w.subs[msg.Channel]
		w.subsMu.Unlock()

		w.applyInExtensions(msg)

		if subscription != nil {
			subscription <- msg
		}
	}
}

func (w *Websocket) Name() string {
	return transportName
}

func (w *Websocket) sendMessage(m *message.Message) error {
	w.applyOutExtensions(m)

	var payload []message.Message
	payload = append(payload, *m)
	return w.conn.WriteJSON(payload)
}
func (w *Websocket) nextMsgID() string {
	return strconv.Itoa(int(atomic.AddUint64(w.msgID, 1)))
}

func (w *Websocket) Options() *transport.Options {
	return w.TransportOpts
}

func (w *Websocket) Handshake() (err error) {
	m := message.Message{
		Channel:                  transport.Handshake,
		Version:                  "1.0", //todo const
		SupportedConnectionTypes: []string{transportName},
	}
	err = w.sendMessage(&m)
	if err != nil {
		return err
	}

	var hsResps []message.Message
	if err = w.conn.ReadJSON(&hsResps); err != nil {
		return err
	}

	resp := &hsResps[0]
	w.applyInExtensions(resp)
	if resp.GetError() != nil {
		return err
	}
	w.clientID = resp.ClientId
	return nil
}

func (w *Websocket) Connect() error {
	m := message.Message{
		Channel:        transport.Connect,
		ClientId:       w.clientID,
		ConnectionType: transportName,
		Id:             w.nextMsgID(),
	}
	//todo expect connect resp from server
	go w.readWorker()
	return w.sendMessage(&m)
}

func (w *Websocket) Subscribe(subscription string, onMessage func(message *message.Message)) error {
	m := &message.Message{
		Channel:      transport.Subscribe,
		ClientId:     w.clientID,
		Subscription: subscription,
		Id:           w.nextMsgID(),
	}

	if err := w.sendMessage(m); err != nil {
		return err
	}

	//todo validate
	inMsgCh := make(chan *message.Message, 0)

	w.subsMu.Lock()
	w.subs[subscription] = inMsgCh
	w.subsMu.Unlock()

	var inMsg *message.Message
	for inMsg = range inMsgCh {
		if inMsg.GetError() != nil {
			return inMsg.GetError()
		}
		onMessage(inMsg)
	}
	return nil
}

func (w *Websocket) Unsubscribe(subscription string) error {
	//https://docs.cometd.org/current/reference/#_bayeux_meta_unsubscribe
	m := &message.Message{
		Channel:      transport.Unsubscribe,
		Subscription: subscription,
		ClientId:     w.clientID,
		Id:           w.nextMsgID(),
	}
	return w.sendMessage(m)
}

func (w *Websocket) Publish(subscription string, message *message.Message) error {
	panic("not implemented")
}

func (w *Websocket) applyOutExtensions(m *message.Message) {
	for i := range w.TransportOpts.OutExt {
		w.TransportOpts.OutExt[i](m)
	}
}
func (w *Websocket) applyInExtensions(m *message.Message) {
	for i := range w.TransportOpts.InExt {
		w.TransportOpts.InExt[i](m)
	}
}

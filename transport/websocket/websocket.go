package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/transport"
	"strconv"
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
	subs          map[string]chan struct{}
}

var _ transport.Transport = (*Websocket)(nil)

func (w *Websocket) Init(options *transport.Options) error {
	var (
		err   error
		msgID uint64
	)
	w.TransportOpts = options
	w.msgID = &msgID
	w.subs = map[string]chan struct{}{}
	w.conn, _, err = websocket.DefaultDialer.Dial(options.Url, nil)
	if err != nil {
		return err
	}
	return nil
}
func (w *Websocket) Name() string {
	return transportName
}
func (w *Websocket) nextMsgID() string {
	return strconv.Itoa(int(atomic.AddUint64(w.msgID, 1)))
}

func (w *Websocket) Options() *transport.Options {
	return w.TransportOpts
}
func (w *Websocket) Handshake() (err error) {
	if err = w.conn.WriteJSON(append([]message.Message{}, message.Message{
		Channel:                  string(transport.Handshake),
		Version:                  "1.0", //todo const
		SupportedConnectionTypes: []string{transportName},
	})); err != nil {
		return err
	}

	var hsResps []message.Message
	if err = w.conn.ReadJSON(&hsResps); err != nil {
		return err
	}

	resp := hsResps[0]
	if resp.GetError() != nil {
		return err
	}
	w.clientID = resp.ClientId
	return nil
}

func (w *Websocket) Connect() error {
	//todo verify if extensions are applied on connect,verify if hs is complete
	return w.conn.WriteJSON(append([]message.Message{}, message.Message{
		Channel:        string(transport.Connect),
		ClientId:       w.clientID,
		ConnectionType: transportName,
		Id:             w.nextMsgID(),
	}))
}

func (w *Websocket) Subscribe(subscription string, onMessage func(message *message.Message)) error {
	m := &message.Message{
		Channel:      string(transport.Subscribe),
		ClientId:     w.clientID,
		Subscription: "/" + subscription,
		Id:           w.nextMsgID(),
	}
	if w.TransportOpts.OutExt != nil {
		w.TransportOpts.OutExt(m)
	}
	err := w.conn.WriteJSON(append([]message.Message{}, *m))
	if err != nil {
		return err
	}

	var hsResps []message.Message
	if err = w.conn.ReadJSON(&hsResps); err != nil {
		return err
	}

	subResp := hsResps[0]
	if subResp.GetError() != nil {
		return err
	}
	if !subResp.Successful {
		//report err just for sanity
	}
	unsubsCh := make(chan struct{}, 0)
	//todo multiple subs
	w.subs[subscription] = unsubsCh

	for {
		select {
		case <-unsubsCh:
			return nil
		default:
		}
		//todo guard unsusribe
		var hsResps []message.Message
		err := w.conn.ReadJSON(&hsResps)
		if err != nil {
			return err
		}

		msg := hsResps[0]
		onMessage(&msg)
	}
	return nil
}

func (w *Websocket) Unsubscribe(subscription string) error {
	panic("not implemented")
}

func (w *Websocket) Publish(subscription string, message *message.Message) error {
	panic("not implemented")
}

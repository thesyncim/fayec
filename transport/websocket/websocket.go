package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/transport"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const transportName = "websocket"

func init() {
	transport.RegisterTransport(&Websocket{})
}

var Debug = true

func debugJson(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", " ")
	return string(b)
}

type Websocket struct {
	TransportOpts *transport.Options
	conn          *websocket.Conn
	clientID      string
	msgID         *uint64

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
	var payload []message.Message
	for {
		err := w.conn.ReadJSON(&payload)
		if err != nil {
			return err
		}
		//dispatch
		msg := payload[0]
		if strings.HasPrefix(msg.Channel, "/meta") {
			continue //todo update introspect message and update state
		}

		w.subsMu.Lock()
		subscription := w.subs[msg.Channel]
		w.subsMu.Unlock()

		subscription <- &msg
	}

}

func (w *Websocket) Name() string {
	return transportName
}

func (w *Websocket) sendMessage(m *message.Message) error {
	var payload []message.Message
	payload = append(payload, *m)
	if Debug {
		log.Println("sending request", debugJson(payload))
	}
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
		Channel:                  string(transport.Handshake),
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
	if Debug {
		log.Println("handshake response", debugJson(hsResps))
	}

	resp := hsResps[0]
	if resp.GetError() != nil {
		return err
	}
	log.Println(debugJson(resp))
	w.clientID = resp.ClientId
	return nil
}

func (w *Websocket) Connect() error {
	m := message.Message{
		Channel:        string(transport.Connect),
		ClientId:       w.clientID,
		ConnectionType: transportName,
		Id:             w.nextMsgID(),
	}
	//todo verify if extensions are applied on connect,verify if hs is complete
	return w.sendMessage(&m)
}

func (w *Websocket) Subscribe(subscription string, onMessage func(message *message.Message)) error {
	m := &message.Message{
		Channel:      string(transport.Subscribe),
		ClientId:     w.clientID,
		Subscription: subscription,
		Id:           w.nextMsgID(),
	}
	if w.TransportOpts.OutExt != nil {
		w.TransportOpts.OutExt(m)
	}

	if err := w.sendMessage(m); err != nil {
		return err
	}

	//todo validate

	inMsgCh := make(chan *message.Message, 0)

	w.subs[subscription] = inMsgCh

	var inMsg *message.Message
	for inMsg = range inMsgCh {
		onMessage(inMsg)
	}
	return nil
}

func (w *Websocket) Unsubscribe(subscription string) error {
	panic("not implemented")
}

func (w *Websocket) Publish(subscription string, message *message.Message) error {
	panic("not implemented")
}

package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/transport"
	"log"
	"sync"
	"sync/atomic"
)

const transportName = "websocket"

func init() {
	transport.RegisterTransport(&Websocket{})
}

//Websocket represents an websocket transport for the faye protocol
type Websocket struct {
	topts *transport.Options

	connMu sync.Mutex
	conn   *websocket.Conn

	advice atomic.Value //type message.Advise //todo move to dispatcher

	stopCh chan error //todo replace wth context

	onMsg           func(msg *message.Message)
	onError         func(err error)
	onTransportDown func(err error)
	onTransportUp   func()
}

var _ transport.Transport = (*Websocket)(nil)

//Init initializes the transport with the provided options
func (w *Websocket) Init(endpoint string, options *transport.Options) error {
	var (
		err error
	)
	w.topts = options

	w.stopCh = make(chan error)
	w.conn, _, err = websocket.DefaultDialer.Dial(endpoint, options.Headers)
	if err != nil {
		return err
	}

	w.conn.SetPingHandler(func(appData string) error {
		return w.conn.WriteJSON(make([]struct{}, 0))
	})
	if err != nil {
		return err
	}
	return nil
}

//Init initializes the transport with the provided options
func (w *Websocket) SetOnErrorHandler(onError func(err error)) {
	w.onError = onError
}

func (w *Websocket) readWorker() error {
	for {
		select {
		case err := <-w.stopCh:
			return err
		default:
		}
		var payload []message.Message
		err := w.conn.ReadJSON(&payload)
		if err != nil {

			return err
		}
		//dispatch
		msg := &payload[0]
		w.onMsg(msg)
	}
}

//name returns the transport name (websocket)
func (w *Websocket) Name() string {
	return transportName
}

func (w *Websocket) SendMessage(m *message.Message) error {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	var payload []message.Message
	payload = append(payload, *m)

again: //todo move this to scheduler
	err := w.conn.WriteJSON(payload)
	if websocket.IsUnexpectedCloseError(err) {
		advise := w.advice.Load().(*message.Advise)
		if advise.Reconnect == message.ReconnectNone {
			return err
		}
		//reconnect
		//we should re-register again subscriptions again
		goto again
	}
	return nil
}

//Options return the transport Options
func (w *Websocket) Options() *transport.Options {
	return w.topts
}

//Handshake initiates a connection negotiation by sending a message to the /meta/handshake channel.
func (w *Websocket) Handshake(msg *message.Message) (resp *message.Message, err error) {
	err = w.SendMessage(msg)

	if err != nil {
		return nil, err
	}

	var hsResps []message.Message
	if err = w.conn.ReadJSON(&hsResps); err != nil {
		return nil, err
	}

	resp = &hsResps[0]
	return resp, nil
}

//Init is called  after a client has discovered the server’s capabilities with a handshake exchange,
//a connection is established by sending a message to the /meta/connect channel
func (w *Websocket) Connect(msg *message.Message) error {
	go func() {
		log.Fatal(w.readWorker())
	}()
	return w.SendMessage(msg)
}

func (w *Websocket) SetOnTransportDownHandler(onTransportDown func(err error)) {
	w.onTransportDown = onTransportDown
}

func (w *Websocket) SetOnTransportUpHandler(onTransportUp func()) {
	w.onTransportUp = onTransportUp
}

//Disconnect closes all subscriptions and inform the server to remove any client-related state.
//any subsequent method call to the client object will result in undefined behaviour.
func (w *Websocket) Disconnect(m *message.Message) error {
	w.stopCh <- nil
	close(w.stopCh)
	return w.SendMessage(m)
}

func (w *Websocket) SetOnMessageReceivedHandler(onMsg func(*message.Message)) {
	w.onMsg = onMsg
}

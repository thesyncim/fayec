package message

import "errors"

type Extension func(message *Message)

type Message struct {
	Channel                  string      `json:"channel,omitempty"`
	Version                  string      `json:"version,omitempty"`
	SupportedConnectionTypes []string    `json:"supportedConnectionTypes,omitempty"`
	ConnectionType           string      `json:"connectionType,omitempty"`
	MinimumVersion           string      `json:"minimumVersion,omitempty"`
	Successful               bool        `json:"successful,omitempty"`
	Ext                      interface{} `json:"ext,omitempty"`
	Id                       string      `json:"id,omitempty"`
	ClientId                 string      `json:"clientId,omitempty"`
	Advice                   Advise      `json:"advice,omitempty"`
	Data                     interface{} `json:"data,omitempty"`
	Timestamp                uint64      `json:"timestamp,omitempty"`
	AuthSuccessful           bool        `json:"authSuccessful,omitempty"`
	Error                    string      `json:"error,omitempty"`
	Subscription             string      `json:"subscription,omitempty"`
}

func (m *Message) GetError() error {
	if m.Error == "" {
		return nil
	}
	return errors.New(m.Error)
}

type Advise struct {
	Reconnect string `json:"reconnect,omitempty"`
	Interval  int64  `json:"interval,omitempty"`
	Timeout   int64  `json:"timeout,omitempty"`
}

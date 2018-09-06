package extensions

import (
	"encoding/json"
	"github.com/thesyncim/faye/message"
	"io"
	"log"
)

func debugJson(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", " ")
	return string(b)
}

type DebugExtension struct {
	in  *log.Logger
	out *log.Logger
}

func NewDebugExtension(out io.Writer) *DebugExtension {
	li := log.New(out, "InMsg", 0)
	lo := log.New(out, "outMsg", 0)
	return &DebugExtension{in: li, out: lo}
}

func (d *DebugExtension) InExtension(m *message.Message) {
	d.in.Println(debugJson(m))
}
func (d *DebugExtension) OutExtension(m *message.Message) {
	d.out.Println(debugJson(m))
}

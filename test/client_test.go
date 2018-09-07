package test

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	. "github.com/thesyncim/faye"
	"github.com/thesyncim/faye/extensions"
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/subscription"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"testing"
	"time"
)

var once sync.Once

var unauthorizedErr = errors.New("500::unauthorized channel")

func setup(t *testing.T) context.CancelFunc {

	//jump to test dir

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	cmd := exec.CommandContext(ctx,
		"npm", "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	var mcancel context.CancelFunc = func() {

		if runtime.GOOS == "windows" {
			exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(cmd.Process.Pid)).Run()
		}
		cancel()
	}

	return mcancel
}

func TestServerSubscribeAndPublish10Messages(t *testing.T) {
	shutdown := setup(t)

	defer shutdown()

	debug := extensions.NewDebugExtension(os.Stdout)

	client, err := NewClient("ws://localhost:8000/faye", WithExtension(debug.InExtension, debug.OutExtension))
	if err != nil {
		t.Fatal(err)
	}

	var delivered int
	var done sync.WaitGroup
	done.Add(10)
	client.OnPublishResponse("/test", func(message *message.Message) {
		if !message.Successful {
			t.Fatalf("failed to send message with id %s", message.Id)
		}
		delivered++
		done.Done()
	})
	var sub *subscription.Subscription
	go func() {
		sub, err = client.Subscribe("/test")
		if err != nil {
			t.Fatal(err)
		}
		err = sub.OnMessage(func(data message.Data) {
			if data != "hello world" {
				t.Fatalf("expecting: `hello world` got : %s", data)
			}
		})
		if err != nil {
			t.Fatal(err)
		}
	}()

	//give some time for setup
	time.Sleep(time.Second)
	for i := 0; i < 10; i++ {
		id, err := client.Publish("/test", "hello world")
		if err != nil {
			t.Fatal(err)
		}
		log.Println(id, i)
	}

	done.Wait()
	err = sub.Unsubscribe()
	if err != nil {
		t.Fatal(err)
	}
	//try to publish one more message
	id, err := client.Publish("/test", "hello world")
	if err != nil {
		t.Fatal(err)
	}
	log.Println(id)
	if delivered != 10 {
		t.Fatal("message received after client unsubscribe")
	}
}

func TestSubscribeUnauthorizedChannel(t *testing.T) {
	shutdown := setup(t)

	defer shutdown()

	debug := extensions.NewDebugExtension(os.Stdout)

	client, err := NewClient("ws://localhost:8000/faye", WithExtension(debug.InExtension, debug.OutExtension))
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Subscribe("/unauthorized")
	if err != nil {
		if err.Error() != unauthorizedErr.Error() {
			t.Fatalf("expecting `500::unauthorized channel` got : `%s`", err.Error())
		}
		return
	}

	t.Fatal("expecting error")

}

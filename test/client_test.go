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
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

var once sync.Once

var unauthorizedErr = errors.New("500::unauthorized channel")

func init() {
	setup(nil)
}

func setup(t *testing.T) context.CancelFunc {

	//jump to test dir

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	cmd := exec.CommandContext(ctx,
		"npm", "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
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

	debug := extensions.NewDebugExtension(os.Stdout)

	client, err := NewClient("ws://localhost:8000/faye", WithExtension(debug.InExtension, debug.OutExtension))
	if err != nil {
		t.Fatal(err)
	}

	var delivered int
	var done sync.WaitGroup
	done.Add(10)

	var sub *subscription.Subscription
	go func() {
		sub, err = client.Subscribe("/test")
		if err != nil {
			t.Fatal(err)
		}
		err = sub.OnMessage(func(channel string, data message.Data) {
			delivered++
			done.Done()
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
		err := client.Publish("/test", "hello world")
		if err != nil {
			t.Fatal(err)
		}

	}

	done.Wait()
	err = sub.Unsubscribe()
	if err != nil {
		t.Fatal(err)
	}
	//try to publish one more message
	err = client.Publish("/test", "hello world")
	if err != nil {
		t.Fatal(err)
	}

	if delivered != 10 {
		t.Fatal("message received after client unsubscribe")
	}
}

func TestSubscribeUnauthorizedChannel(t *testing.T) {

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

func TestWildcardSubscription(t *testing.T) {

	debug := extensions.NewDebugExtension(os.Stdout)

	client, err := NewClient("ws://localhost:8000/faye", WithExtension(debug.InExtension, debug.OutExtension))
	if err != nil {
		t.Fatal(err)
	}

	var done sync.WaitGroup
	done.Add(20)

	sub, err := client.Subscribe("/wildcard/**")
	if err != nil {
		t.Fatal(err)
	}

	var recv = map[string]int{}

	go func() {
		err := sub.OnMessage(func(channel string, msg message.Data) {
			done.Done()
			recv[channel]++
		})
		if err != nil {
			t.Fatal(err)
		}
	}()

	//give some time for setup
	time.Sleep(time.Second)

	for _, channel := range []string{"/wildcard/foo", "/wildcard/bar"} {
		for i := 0; i < 10; i++ {
			err := client.Publish(channel, "hello world")
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	done.Wait()

	expected := map[string]int{
		"/wildcard/foo": 10,
		"/wildcard/bar": 10}

	if !reflect.DeepEqual(recv, expected) {
		t.Fatal(recv)
	}

}

package logger

import (
	"encoding/json"
	"github.com/g8os/core.base/pm/core"
	"github.com/g8os/core.base/pm/stream"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func getFakeCmd(t *testing.T) *core.Command {
	cmd, err := core.LoadCmd([]byte("{}"))
	if err != nil {
		t.Error("Could not create fake command")
	}
	cmd.ID = "test-id"
	return cmd
}

func TestACLogger_BatchSizeTrigger(t *testing.T) {
	var wg sync.WaitGroup

	serverMux := http.NewServeMux()
	server := &http.Server{
		Handler: serverMux,
	}

	listner, err := net.Listen("tcp", ":1234")
	if err != nil {
		t.Error(err)
	}

	signal := make(chan int)
	handle := func(writer http.ResponseWriter, request *http.Request) {
		defer listner.Close()
		defer request.Body.Close()
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}

		//content is the serialized log messages.
		var messages []*stream.Message
		err = json.Unmarshal(body, &messages)
		if err != nil {
			t.Error(err)
		}
		signal <- len(messages)
	}

	wg.Add(1)
	//starting proxy
	go func() {
		defer wg.Done()
		serverMux.HandleFunc("/logs", handle)
		go server.Serve(listner)
	}()

	//wait until proxy is ready before starting agents.
	wg.Wait()

	logger := NewACLogger(map[string]*http.Client{
		"http://localhost:1234/logs": &http.Client{},
	}, 2, 60*time.Minute, []int{1, 2})

	message1 := "Hello world"

	cmd := getFakeCmd(t)
	msg1 := &stream.Message{
		ID:      1,
		Level:   1,
		Message: message1,
		Epoch:   1000,
	}

	msg2 := &stream.Message{
		ID:      2,
		Level:   1,
		Message: message1,
		Epoch:   1000,
	}

	logger.Log(cmd, msg1)
	logger.Log(cmd, msg2)

	select {
	case l := <-signal:
		if l != 2 {
			t.Error("Invalid number of messages logged")
		}
	case <-time.After(10 * time.Second):
		t.Error("Something went wrong, messages never received at the end point")
	}
}

func TestACLogger_FlushIntTrigger(t *testing.T) {
	var wg sync.WaitGroup

	serverMux := http.NewServeMux()
	server := &http.Server{
		Handler: serverMux,
	}

	listner, err := net.Listen("tcp", ":1236")
	if err != nil {
		t.Error(err)
	}

	signal := make(chan int)
	handle := func(writer http.ResponseWriter, request *http.Request) {
		defer listner.Close()
		defer request.Body.Close()
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}

		//content is the serialized log messages.
		var messages []*stream.Message
		err = json.Unmarshal(body, &messages)
		if err != nil {
			t.Error(err)
		}

		signal <- len(messages)
	}

	wg.Add(1)
	//starting proxy
	go func() {
		defer wg.Done()
		serverMux.HandleFunc("/logs", handle)
		go server.Serve(listner)
	}()

	//wait until proxy is ready before starting agents.
	wg.Wait()

	logger := NewACLogger(map[string]*http.Client{
		"http://localhost:1236/logs": &http.Client{},
	}, 100, 5*time.Second, []int{1, 2})

	message1 := "Hello world"

	cmd := getFakeCmd(t)
	msg1 := &stream.Message{
		ID:      1,
		Level:   1,
		Message: message1,
		Epoch:   1000,
	}

	msg2 := &stream.Message{
		ID:      2,
		Level:   1,
		Message: message1,
		Epoch:   1000,
	}

	msg3 := &stream.Message{
		ID:      2,
		Level:   5,
		Message: message1,
		Epoch:   1000,
	}

	logger.Log(cmd, msg1)
	logger.Log(cmd, msg2)
	logger.Log(cmd, msg3)

	select {
	case l := <-signal:
		if l != 2 {
			t.Error("Invalid number of messages logged")
		}
	case <-time.After(10 * time.Second):
		t.Error("Something went wrong, messages never received at the end point")
	}
}

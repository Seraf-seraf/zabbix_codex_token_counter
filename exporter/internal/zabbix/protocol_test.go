package zabbix

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"
)

func TestEncodeFrame(t *testing.T) {
	now := time.Unix(100, 0)
	request, err := newSenderRequest([]Metric{{Host: "host", Key: "key", Value: int64(12), Timestamp: time.Unix(99, 0)}}, now)
	if err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	if err := writeMessage(&buffer, request); err != nil {
		t.Fatal(err)
	}
	frame := buffer.Bytes()
	if !bytes.Equal(frame[:5], frameHeader) {
		t.Fatalf("bad header: %q", frame[:5])
	}
	length := binary.LittleEndian.Uint64(frame[5:13])
	if int(length) != len(frame)-13 {
		t.Fatalf("bad payload length: %d", length)
	}
	var body map[string]any
	if err := json.Unmarshal(frame[13:], &body); err != nil {
		t.Fatal(err)
	}
	if body["request"] != "sender data" || body["clock"].(float64) != 100 {
		t.Fatalf("bad JSON body: %+v", body)
	}
}

func TestDecodeResponse(t *testing.T) {
	frame := responseFrame(t, `{"response":"success","info":"processed: 2; failed: 0; total: 2; seconds spent: 0.1"}`)
	var response senderResponse
	if err := readMessage(bytes.NewReader(frame), &response); err != nil {
		t.Fatal(err)
	}
	response.parseCounts()
	if response.Processed != 2 || response.Failed != 0 || response.Total != 2 {
		t.Fatalf("bad response: %+v", response)
	}
}

func TestClientWithFakeZabbixServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	received := make(chan senderRequest, 1)
	serverErr := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()
		header := make([]byte, 13)
		if _, err := io.ReadFull(connection, header); err != nil {
			serverErr <- err
			return
		}
		payload := make([]byte, binary.LittleEndian.Uint64(header[5:13]))
		if _, err := io.ReadFull(connection, payload); err != nil {
			serverErr <- err
			return
		}
		var request senderRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			serverErr <- err
			return
		}
		received <- request
		_, err = connection.Write(responseFrame(t, `{"response":"success","info":"processed: 1; failed: 0; total: 1; seconds spent: 0.1"}`))
		serverErr <- err
	}()

	client := NewClient(listener.Addr().String(), time.Second)
	if err := client.Send(context.Background(), []Metric{{Host: "host", Key: "key", Value: true}}); err != nil {
		t.Fatal(err)
	}
	request := <-received
	if len(request.Data) != 1 || request.Data[0].Value != "true" {
		t.Fatalf("bad request: %+v", request)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestClientRejectsFailedMetrics(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		connection, _ := listener.Accept()
		defer connection.Close()
		header := make([]byte, 13)
		_, _ = io.ReadFull(connection, header)
		_, _ = io.CopyN(io.Discard, connection, int64(binary.LittleEndian.Uint64(header[5:13])))
		_, _ = connection.Write(responseFrame(t, `{"response":"success","info":"processed: 0; failed: 1; total: 1"}`))
	}()
	err = NewClient(listener.Addr().String(), time.Second).Send(context.Background(), []Metric{{Host: "host", Key: "key", Value: "x"}})
	if err == nil {
		t.Fatal("expected failed metric error")
	}
}

func responseFrame(t *testing.T, payload string) []byte {
	t.Helper()
	frame := make([]byte, 13+len(payload))
	copy(frame, frameHeader)
	binary.LittleEndian.PutUint64(frame[5:13], uint64(len(payload)))
	copy(frame[13:], payload)
	return frame
}

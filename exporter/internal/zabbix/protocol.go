package zabbix

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"
)

const maxMessageSize = 16 * 1024 * 1024

var (
	frameHeader = []byte("ZBXD\x01")
	infoCounts  = regexp.MustCompile(`processed:\s*(\d+);\s*failed:\s*(\d+);\s*total:\s*(\d+)`)
)

type wireMetric struct {
	Host  string `json:"host"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Clock int64  `json:"clock,omitempty"`
}

type senderRequest struct {
	Request string       `json:"request"`
	Data    []wireMetric `json:"data"`
	Clock   int64        `json:"clock"`
}

type senderResponse struct {
	Response  string `json:"response"`
	Info      string `json:"info"`
	Processed int    `json:"processed,omitempty"`
	Failed    int    `json:"failed,omitempty"`
	Total     int    `json:"total,omitempty"`
}

func newSenderRequest(metrics []Metric, now time.Time) (senderRequest, error) {
	data := make([]wireMetric, 0, len(metrics))
	for _, item := range metrics {
		value, err := FormatValue(item.Value)
		if err != nil {
			return senderRequest{}, fmt.Errorf("метрика %q: %w", item.Key, err)
		}
		clock := item.Timestamp.Unix()
		if item.Timestamp.IsZero() {
			clock = now.Unix()
		}
		data = append(data, wireMetric{Host: item.Host, Key: item.Key, Value: value, Clock: clock})
	}
	return senderRequest{Request: "sender data", Data: data, Clock: now.Unix()}, nil
}

func writeMessage(writer io.Writer, message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать сообщение: %w", err)
	}
	frame := make([]byte, len(frameHeader)+8+len(payload))
	copy(frame, frameHeader)
	binary.LittleEndian.PutUint64(frame[len(frameHeader):], uint64(len(payload)))
	copy(frame[len(frameHeader)+8:], payload)
	if _, err := io.Copy(writer, bytes.NewReader(frame)); err != nil {
		return fmt.Errorf("не удалось записать сообщение: %w", err)
	}
	return nil
}

func readMessage(reader io.Reader, message any) error {
	header := make([]byte, len(frameHeader)+8)
	if _, err := io.ReadFull(reader, header); err != nil {
		return fmt.Errorf("не удалось прочитать заголовок сообщения: %w", err)
	}
	if !bytes.Equal(header[:len(frameHeader)], frameHeader) {
		return fmt.Errorf("некорректный заголовок сообщения Zabbix")
	}
	length := binary.LittleEndian.Uint64(header[len(frameHeader):])
	if length > maxMessageSize {
		return fmt.Errorf("сообщение Zabbix слишком велико: %d байт", length)
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(reader, payload); err != nil {
		return fmt.Errorf("не удалось прочитать содержимое сообщения: %w", err)
	}
	if err := json.Unmarshal(payload, message); err != nil {
		return fmt.Errorf("не удалось декодировать сообщение: %w", err)
	}
	return nil
}

func (response *senderResponse) parseCounts() {
	if matches := infoCounts.FindStringSubmatch(response.Info); len(matches) == 4 {
		response.Processed, _ = strconv.Atoi(matches[1])
		response.Failed, _ = strconv.Atoi(matches[2])
		response.Total, _ = strconv.Atoi(matches[3])
	}
}

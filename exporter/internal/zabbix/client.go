package zabbix

import (
	"context"
	"fmt"
	"net"
	"time"
)

type Sender interface {
	Send(context.Context, []Metric) error
}

type Client struct {
	address string
	timeout time.Duration
	dialer  net.Dialer
}

func NewClient(address string, timeout time.Duration) *Client {
	return &Client{address: address, timeout: timeout, dialer: net.Dialer{Timeout: timeout}}
}

func (c *Client) Send(ctx context.Context, metrics []Metric) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(metrics) == 0 {
		return nil
	}

	request, err := newSenderRequest(metrics, time.Now())
	if err != nil {
		return err
	}
	connection, err := c.dialer.DialContext(ctx, "tcp", c.address)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к Zabbix по адресу %s: %w", c.address, err)
	}
	defer connection.Close()

	deadline := time.Now().Add(c.timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return fmt.Errorf("не удалось установить срок ожидания соединения с Zabbix: %w", err)
	}
	if err := writeMessage(connection, request); err != nil {
		return fmt.Errorf("не удалось записать запрос Zabbix: %w", err)
	}
	var response senderResponse
	if err := readMessage(connection, &response); err != nil {
		return fmt.Errorf("не удалось прочитать ответ Zabbix: %w", err)
	}
	response.parseCounts()
	if response.Response != "success" {
		return fmt.Errorf("zabbix отклонил данные отправителя: response=%q info=%q", response.Response, response.Info)
	}
	if response.Failed > 0 {
		return fmt.Errorf("zabbix не обработал часть метрик: processed=%d failed=%d total=%d info=%q", response.Processed, response.Failed, response.Total, response.Info)
	}
	if response.Total > 0 && response.Total != len(metrics) {
		return fmt.Errorf("общее количество метрик в ответе Zabbix не совпадает: sent=%d total=%d info=%q", len(metrics), response.Total, response.Info)
	}
	return nil
}

// Package rabbitmq wraps amqp091-go to provide a small, opinionated
// Publisher/Consumer pair used by all services for domain events.
package rabbitmq

import (
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Config captures the connection parameters for a RabbitMQ broker.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	VHost    string
}

// URI renders the amqp:// connection URI.
func (c Config) URI() string {
	vhost := c.VHost
	if vhost == "" {
		vhost = "/"
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s", c.User, c.Password, c.Host, c.Port, vhost)
}

// Publisher publishes messages to an exchange with a routing key.
type Publisher struct {
	cfg       Config
	mu        sync.Mutex
	conn      *amqp.Connection
	ch        *amqp.Channel
	exchange  string
	durable   bool
	connected bool
}

// NewPublisher constructs a new Publisher for the given exchange.
// The underlying connection is opened lazily on the first Publish call so
// that misconfigured brokers do not break service startup.
func NewPublisher(cfg Config, exchange string, durable bool) *Publisher {
	return &Publisher{cfg: cfg, exchange: exchange, durable: durable}
}

// ensure opens a connection + channel and declares the exchange.
func (p *Publisher) ensure() error {
	if p.connected && p.conn != nil && !p.conn.IsClosed() {
		return nil
	}
	// close any stale handle.
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}

	conn, err := amqp.DialConfig(p.cfg.URI(), amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("rabbitmq channel: %w", err)
	}
	if err := ch.ExchangeDeclare(p.exchange, "topic", p.durable, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("declare exchange %s: %w", p.exchange, err)
	}
	p.conn = conn
	p.ch = ch
	p.connected = true
	return nil
}

// Publish a message body with the given routing key.
func (p *Publisher) Publish(routingKey, contentType string, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensure(); err != nil {
		return err
	}
	return p.ch.Publish(p.exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  contentType,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         body,
	})
}

// Close releases the underlying channel and connection.
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
	p.connected = false
	return nil
}

// ErrNotConnected is returned when a publish is attempted before the broker
// is reachable.
var ErrNotConnected = errors.New("rabbitmq: not connected")

package caddyeventskafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyevents"
	kafka "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
	"go.uber.org/zap"
)

type SASLAlgorithm string

var (
	SASLMethodSHA256 SASLAlgorithm = "sha256"
	SASLMethodSHA512 SASLAlgorithm = "sha512"
)

type KafkaHandler struct {
	logger *zap.Logger
	writer *kafka.Writer

	BootstrapServers []string      `json:"bootstrap_servers"`
	Topic            string        `json:"topic"`
	TLSEnabled       bool          `json:"tls_enabled"`
	TLSNoVerify      bool          `json:"tls_no_verify"`
	SASLAuth         bool          `json:"sasl_auth"`
	SASLAlgorithm    SASLAlgorithm `json:"sasl_algorithm"`
	SASLUsername     string        `json:"sasl_username"`
	SASLPassword     string        `json:"sasl_password"`
}

func init() {
	caddy.RegisterModule(KafkaHandler{})
}

func (h *KafkaHandler) Provision(ctx caddy.Context) error {
	h.logger = ctx.Logger(h)

	h.writer = &kafka.Writer{
		Addr:  kafka.TCP(h.BootstrapServers...),
		Async: true,
	}

	transport := &kafka.Transport{}
	h.writer.Transport = transport

	if h.TLSEnabled {
		tls_config := &tls.Config{}

		if h.TLSNoVerify {
			tls_config.InsecureSkipVerify = true
		}

		transport.TLS = tls_config
	}

	if h.SASLAuth {
		var method scram.Algorithm
		if h.SASLAlgorithm == SASLMethodSHA256 {
			method = scram.SHA256
		} else if h.SASLAlgorithm == SASLMethodSHA512 {
			method = scram.SHA512
		} else {
			return fmt.Errorf("invalid sasl algorithm: %s", h.SASLAlgorithm)
		}

		mechanism, err := scram.Mechanism(method, h.SASLUsername, h.SASLPassword)
		if err != nil {
			panic(err)
		}
		transport.SASL = mechanism
	}

	return nil
}

func (KafkaHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "events.handlers.kafka",
		New: func() caddy.Module {
			return new(KafkaHandler)
		},
	}
}

func (h *KafkaHandler) Validate() error {
	if h.BootstrapServers == nil || len(h.BootstrapServers) == 0 {
		return fmt.Errorf("kafka bootstrap_servers missing")
	}

	if h.Topic == "" {
		return fmt.Errorf("kafka topic missing")
	}

	if h.SASLAuth {
		if h.SASLUsername == "" {
			return fmt.Errorf("kafka missing sasl username")
		}
		if h.SASLPassword == "" {
			return fmt.Errorf("kafka missing sasl username")
		}
		if h.SASLAlgorithm != "sha256" && h.SASLAlgorithm != "sha512" {
			return fmt.Errorf("kafka invalid sasl algorithm")
		}
	}

	return nil
}

func (h *KafkaHandler) Handle(ctx context.Context, e caddyevents.Event) error {
	data, err := json.Marshal(e.CloudEvent())
	if err != nil {
		return fmt.Errorf("failed to serialize event data: %v", err)
	}

	if err := h.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(e.CloudEvent().ID),
		Value: data,
		Topic: h.Topic,
		Time:  e.CloudEvent().Time,
	}); err != nil {
		return fmt.Errorf("failed to write event to Kafka: %v", err)
	}

	return nil
}

func (h *KafkaHandler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	if !d.Next() {
		return d.ArgErr()
	}
	for d.NextBlock(0) {
		switch val := d.Val(); val {
		case "topic":
			if !d.Args(&h.Topic) {
				return d.ArgErr()
			}
		case "sasl_scram":
			var mech string
			if !d.Args(&mech, &h.SASLUsername, &h.SASLPassword) {
				return d.ArgErr()
			}

			if mech == "sha256" {
				h.SASLAlgorithm = SASLMethodSHA256
			} else if mech == "sha512" {
				h.SASLAlgorithm = SASLMethodSHA512
			} else {
				return fmt.Errorf("unsupported SCRAM method: %s", mech)
			}

			h.SASLAuth = true
		case "tls":
			tls_enabled := ""
			if !d.Args(&tls_enabled) {
				return d.ArgErr()
			}

			if tls_enabled == "on" {
				h.TLSEnabled = true
			} else if tls_enabled == "off" {
				h.TLSEnabled = false
			} else {
				return fmt.Errorf("invalid value for tls")
			}
		case "tls_no_verify":
			h.TLSNoVerify = true
		case "bootstrap_servers":
			if !d.Next() {
				return d.ArgErr()
			}

			h.BootstrapServers = make([]string, 0)
			items := strings.Split(d.Val(), ",")
			for _, item := range items {
				h.BootstrapServers = append(h.BootstrapServers, strings.TrimSpace(item))
			}
		default:
			return fmt.Errorf("unsupported argument: %s", val)
		}
	}

	return nil
}

var (
	_ caddyfile.Unmarshaler = (*KafkaHandler)(nil)
	_ caddy.Provisioner     = (*KafkaHandler)(nil)
	_ caddyevents.Handler   = (*KafkaHandler)(nil)
	_ caddy.Validator       = (*KafkaHandler)(nil)
)

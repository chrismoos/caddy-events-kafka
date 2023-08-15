# Caddy Kafka Events Handler Module

This module implements an event handler for Caddy that publishes events to a Kafka topic. This is useful if you want to consume events (for example, certificate lifecyle events) in an external system.

## Kafka Messages

Messages published to Kafka have a `String` key, with a unique UUID for the event (from `id` below). The value is a JSON object of the `CloudEvent` type from Caddy. For example an event produced when a certificate is obtained:

```json
{
    "id": "7589856b-df84-49d1-91d7-2dc3b6c5eded",
    "source": "tls",
    "specversion": "1.0",
    "type": "cert_obtained",
    "time": "2023-08-14T18:09:35.984688502Z",
    "datacontenttype": "application/json",
    "data": {
        "certificate_path": "certificates/acme.zerossl.com-v2-dv90/www.exampledomain.com/www.exampledomain.com.crt",
        "identifier": "www.exampledomain.com",
        "issuer": "acme.zerossl.com-v2-DV90",
        "metadata_path": "certificates/acme.zerossl.com-v2-dv90/www.exampledomain.com/www.exampledomain.com.json",
        "private_key_path": "certificates/acme.zerossl.com-v2-dv90/www.exampledomain.com/www.exampledomain.com.key",
        "renewal": false,
        "storage_path": "certificates/acme.zerossl.com-v2-dv90/www.exampledomain.com"
    }
}

```

## Installation

You can use [xcaddy](https://github.com/caddyserver/xcaddy) to build Caddy with this module.

```sh
$ xcaddy build --with github.com/chrismoos/caddy-events-kafka
```

## Configuration

The module supports configuration via Caddyfile or JSON.

## Caddyfile

|Argument|Description|Example|
|---|---|---|
|bootstrap_servers|Comma separated list of bootstrap servers for Kafka|127.0.0.1:9092|
|topic|The Kafka topic to publish events.|caddy-events|
|tls|`on` to enable TLS, or `off` (default) to disable|tls on|
|tls_no_verify|**DANGER**: If present, disables TLS peer verification, this should only be used for testing and not production. defaults to off (TLS verification enabled)||
|sasl_scram|If specified, enables SASL authentication. The mechanism (`sha256` or `sha512`) must be specified, as well as a username and password.|sasl_scram sha512 "myuser" "mypassword"

## JSON


```json
{
    "bootstrap_servers": ["127.0.0.1:9092"],
    "topic": "caddy-events",
    "tls_enabled": true,
    "tls_no_verify": false,
    "sasl_auth": true,
    "sasl_username": "myuser",
    "sasl_password": "mypassword"
}
```

## Caddyfile Examples

#### TLS + SASL Authentication

```caddyfile
{
	events {
		on "*" kafka {
			bootstrap_servers "localhost:9096"
			topic "caddy-events"
			tls on
			sasl_scram sha512 "sasl_user" "sasl_password"
		}
	}
}

localhost
respond "Hello"
```

#### Plaintext

```caddyfile
{
	events {
		on "*" kafka {
			bootstrap_servers "localhost:29092"
			topic "caddy-events"
		}
	}
}

localhost
respond "Hello"
```

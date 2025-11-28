[![Go Report Card](https://goreportcard.com/badge/github.com/dimiro1/ipe)](https://goreportcard.com/report/github.com/dimiro1/ipe)

Try browsing [the code on Sourcegraph](https://sourcegraph.com/github.com/dimiro1/ipe)!

# IPÊ

An open source Pusher server implementation compatible with Pusher client libraries written in Go.

# Why I wrote this software?

1. I wanted to learn Go and I needed a non trivial application;
2. I use Pusher in some projects;
3. I really like Pusher;
4. I was using Pusher on some projects behind a firewall;

# Features

* Public Channels;
* Private Channels;
* Presence Channels;
* Web Hooks;
* Client events;
* Complete REST API;
* Easy installation;
* A single binary without dependencies;
* Easy configuration;
* Protocol version 7;
* Multiple apps in the same instance;
* Drop in replacement for pusher server;
* Horizontal scaling with Redis;

# Horizontal Scaling with Redis

IPÊ uses Redis for cross-instance message distribution, enabling horizontal scaling across multiple pods or instances. This allows you to run multiple IPÊ instances behind a load balancer while ensuring all connected clients receive messages regardless of which instance they're connected to.

## How it works

When a client connects to one instance (e.g., Pod A) and subscribes to a channel, the subscription is registered both locally and in Redis. If a message is published to a different instance (e.g., Pod B), that instance:

1. Publishes the message to its local connections
2. Detects that there are subscribers on other instances via Redis
3. Publishes the message to Redis Pub/Sub
4. All instances subscribed to that channel receive the message via Redis
5. Each instance delivers the message to its local connections

This ensures that clients connected to any instance will receive messages published from any other instance, making IPÊ suitable for production deployments requiring high availability and scalability.

**Note:** Redis is required for the server to start. The application will fail to initialize if it cannot connect to Redis.

# Download pre built binaries

You can download pre built binaries from the [releases tab](https://github.com/dimiro1/ipe/releases).

# Building

```console
$ go get github.com/dimiro1/ipe
```

or simply

```console
$ go install github.com/dimiro1/ipe
```

Building from a local checkout

```console
$ git clone https://github.com/dimiro1/ipe.git
$ cd ipe/cmd
$ go build -o ipe
```

# How to configure?

## The server

```yaml

---
host: ":8080"
profiling: false
ssl:
  enabled: false
  host: ":4343"
  key_file: "key.pem"
  cert_file: "cert.pem"
apps:
  - name: "Sample Application"
    enabled: true
    only_ssl: false
    key: "278d525bdf162c739803"
    secret: "${APP_SECRET}" # Expand env vars
    app_id: "1"
    user_events: true
    webhooks:
      enabled: true
      url: "http://127.0.0.1:5000/hook"

```

## Redis Configuration

IPÊ requires Redis to be configured and running. Redis is used for:
- Cross-instance message distribution via Pub/Sub
- Connection and subscription metadata storage
- Presence channel data synchronization

### Environment Variables

Redis is configured using the following environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_HOST` | `localhost` | Redis server hostname or IP address |
| `REDIS_PORT` | `6379` | Redis server port |
| `REDIS_PASSWORD` | (empty) | Redis password (if required) |
| `REDIS_DB` | `0` | Redis database number (ignored in cluster mode) |
| `REDIS_CLUSTER_MODE` | `false` | Set to `true` to enable Redis Cluster mode |
| `REDIS_ADDRS` | (empty) | Comma-separated list of cluster node addresses (optional) |
| `REDIS_TLS_ENABLED` | `false` | Set to `true` to enable TLS/SSL connections |
| `IPE_INSTANCE_ID` | (auto-generated) | Unique identifier for this instance (optional) |

### Example Configuration

For local development (standalone):

```console
$ export REDIS_HOST=localhost
$ export REDIS_PORT=6379
$ ./ipe -config config.yml
```

For production with authentication (standalone):

```console
$ export REDIS_HOST=redis.example.com
$ export REDIS_PORT=6379
$ export REDIS_PASSWORD=your-secure-password
$ export REDIS_DB=0
$ export IPE_INSTANCE_ID=pod-1
$ ./ipe -config config.yml
```

For Redis Cluster (e.g., AWS ElastiCache):

```console
$ export REDIS_HOST=clustercfg.your-cluster.region.cache.amazonaws.com
$ export REDIS_PORT=6379
$ export REDIS_CLUSTER_MODE=true
$ export REDIS_TLS_ENABLED=true
$ export REDIS_PASSWORD=your-auth-token
$ ./ipe -config config.yml
```

### Quick Start with Docker

For local development, you can quickly start Redis using Docker:

```console
$ docker run -d -p 6379:6379 --name ipe-redis redis:latest
$ export REDIS_HOST=localhost
$ export REDIS_PORT=6379
$ ./ipe -config config.yml
```

### Production Considerations

- **High Availability**: Use Redis Sentinel or Redis Cluster for production deployments
- **Instance IDs**: Set unique `IPE_INSTANCE_ID` values for each pod/instance (e.g., using Kubernetes pod names or environment-specific identifiers)
- **Connection Pooling**: The Redis client handles connection pooling automatically
- **Data Expiration**: Connection and subscription metadata expires after 24 hours of inactivity

## Libraries

### Client javascript library

```javascript
let pusher = new Pusher(APP_KEY, {
  wsHost: 'localhost',
  wsPort: 8080,
  wssPort: 4433,    // Required if encrypted is true
  encrypted: false, // Optional. the application must use only SSL connections
  enabledTransports: ["ws", "flash"],
  disabledTransports: ["flash"]
});
```

### Client server libraries

Ruby

```ruby
Pusher.host = 'localhost'
Pusher.port = 8080
```

PHP

```php
$pusher = new Pusher(APP_KEY, APP_SECRET, APP_ID, DEBUG, "http://localhost", "8080");
```

Go

```go
package main

import "github.com/pusher/pusher-http-go"

func main() {
	client := pusher.Client{
        AppId:  "APP_ID",
        Key:    "APP_KEY",
        Secret: "APP_SECRET",
        Host:   ":8080",
    }
	
	// use the client
}
```

NodeJS

```javascript
let pusher = new Pusher({
  appId: APP_ID,
  key: APP_KEY,
  secret: APP_SECRET
  domain: 'localhost',
  port: 80
});

```

# Logging

This software uses the [glog](https://github.com/golang/glog) library

for more information about logging type the following in console.

```console
$ ipe -h
```

# When use this software?

* When you are offline;
* When you want to control your infrastructure;
* When you do not want to have external dependencies;
* When you want extend the protocol;
* When you need scalable multi-instance deployments with horizontal scaling;

# Contributing.

Feel free to fork this repo.

# Pusher

Pusher is an excellent service, their service is very reliable. I recommend for everyone.

# Where this name came from?

Here in Brazil we have this beautiful tree called [Ipê](http://en.wikipedia.org/wiki/Tabebuia_aurea), it comes in differente colors: yellow, pink, white, purple.

[I want to see pictures](https://www.flickr.com/search/?q=ipe)

# Author

Claudemiro Alves Feitosa Neto

# LICENSE

Copyright 2014, 2018 Claudemiro Alves Feitosa Neto. All rights reserved.
Use of this source code is governed by a MIT-style
license that can be found in the LICENSE file.


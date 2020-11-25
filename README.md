<div align="center">
  <br/>
  <img src="./ethlogspy.png" width="200" />
  <br/>
  <br/>
  <p>
    Reverse proxy for Ethereum nodes that stores logs information for a faster retrieval.
  </p>
  <p>
    version 1.0.0-beta
  </p>
  <br/>
  <p>
    <a href="#status"><strong>Status</strong></a> ·
    <a href="#description"><strong>Description</strong></a> ·
    <a href="#features"><strong>Features</strong></a> ·
    <a href="#install"><strong>Install</strong></a> ·
    <a href="#contributing"><strong>Contributing</strong></a>
  </p>
</div>

---

## Status

**EthLogSpy** is currently in **beta** version.

---

## Description

**EthLogSpy** is a Golang Reverse Proxy for an Ethereum Node.

It exposes the given node to the world by just reverse proxying every JSON RPC call, except for the `eth_getLogs` one: on this method EthLogSpy retrieves the logs stored in the database (**MongoDB**) or in the cache (**Redis**) and returns it in the same format that the Ethereum Node would use, basically faking its response.

The yaml configuration file is layed out as follows:

```yaml
mongo:
  connection: "mongodb://localhost:27017" # mongodb connection string
  db_name: "ethlogspy" # mongodb database name
node:
  host: "localhost" # ethereum node host
  port: 8535 # ethereum node port
redis:
  host: "localhost" # redis host 
  port: 6379 # redis port
  password: "" # redis password
  db: 0 # redis db
server:
  block_number: 0 # starting block number for log syncing
```

---

## Features

- [x] HTTP JSON RPC API;
- [x] WebSocket JSON RPC API;
- [x] Support for `eth_subscribe` JSON RPC method. 

---

## Install

### Docker

In order to install `ethlogspy` using Docker you need to have a running **MongoDB** and **Redis** instance. After you've cloned this repo, you need to go and update your desired config in the `configs/` folder based on your environment. 

Finally, build the docker image and run it:

```
docker build -t ethlogspy .
docker run -it --rm -d -p 8080:8080 --name ethlogspy ethlogspy:latest
```

If you're also running MongoDB and Redis on a container, make sure to link them to the `ethlogspy` one. If your Ethereum Node is running locally, make sure also to allow the `ethlogspy` container to contact it by using the proper Docker configuration.

For **MacOS** and **Windows** just use the `host.docker.internal` instead of `localhost` in the configuration to reach your node; on **Linux** you need to update the `docker run` command and add the host network flag (`--network="host"`).

### Docker Compose

If you want to run everyting using docker-compose, you simply just clone this repo and then run inside the folder:

```
docker-compose build
nohup docker-compose up &
```

This will build and run a local instance of MongoDB, Redis and EthLogSpy, ready for use.

---

## Contributing

We welcome community contributions!

Please check out our <a href="https://github.com/PaoloRollo/ethlogspy/issues">open issues</a> to get started.

If you discover something that could potentially impact security, please notify us immediately by sending an e-mail at <a href="mailto:paolo.rollo1997@gmail.com">paolo.rollo1997@gmail.com</a>. We'll get in touch with you as fast as we can!

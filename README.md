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
    <a href="#example"><strong>Example</strong></a> ·
    <a href="#security"><strong>Security</strong></a> ·
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

It is fully compatible with the `web3js` library, so your dApp can easily retrieve the logs just by using the `web3` instance (thus avoiding making any specific HTTP request to EthLogSpy).

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
- [x] Support for `eth_subscribe` JSON RPC method;
- [ ] Disable/enable Redis cache via configuration;
- [ ] Disable/enable in-memory cache via configuration;
- [ ] Logs integrity check (via routine or after every call?).

---

## Install

### Standalone

If you have `golang` installed on your machine you can simply build the executable by running the following command inside the cloned directory:

```bash
go build -o ethlogspy *.go
```

This will create the `ethlogspy` executable that you can run using this commands:
```bash
./ethlogspy # default configuration file must be found in /usr/local/ethlogspy/configs/config.yml
./ethlogspy -c CONFIG_PATH # retrieves the config file from the given CONFIG_PATH
```

### Docker

In order to install `ethlogspy` using Docker you need to have a running **MongoDB** and **Redis** instance. After you've cloned this repo, you need to go and update your desired config in the `configs/` folder based on your environment. 

Finally, build the docker image and run it:

```bash
docker build -t ethlogspy .
docker run -it --rm -d -p 8080:8080 --name ethlogspy ethlogspy:latest
```

If you're also running MongoDB and Redis on a container, make sure to link them to the `ethlogspy` one. If your Ethereum Node is running locally, make sure also to allow the `ethlogspy` container to contact it by using the proper Docker configuration.

For **MacOS** and **Windows** just use the `host.docker.internal` instead of `localhost` in the configuration to reach your node; on **Linux** you need to update the `docker run` command and add the host network flag (`--network="host"`).

### Docker Compose

If you want to run everyting using docker-compose, you simply just clone this repo and then run inside the folder:

```bash
docker-compose build
nohup docker-compose up &
```

This will build and run a local instance of MongoDB, Redis and EthLogSpy, ready for use.

---

## Example

These are just few examples of how you easily you can use **web3js** in conjunction with a running **EthLogSpy** instance:

```javascript
const Web3 = require('web3');

const web3 = new Web3("http://localhost:8545"); // no ethlogspy and node on 8545
const web3 = new Web3("http://localhost:8080"); // with ethlogspy on 8080 pointing to node on 8545

const result = await web3.eth.getPastLogs({ fromBlock: 0 }); // retrieve the logs
```

Having a EthLogSpy instance in front of your node doesn't mean, of course, that you can't use all the other RPC APIs:

```javascript
const accounts = await web3.eth.getAccounts(); // can you retrieve the accounts? yes.
const receipt = await web3.eth.sendTransaction(accounts[0], accounts[1], 1000000000); // can you send a transaction? yes.
const contract = new web3.eth.Contract(jsonInterface, address); // can you get a contract? yes.
```

What kind of spy doesn't allow all of this?

---

## Security considerations

Please, if you're running EthLogSpy with MongoDB and Redis on the same machine, do not expose to the public the `27017` and `6379` ports (or whatever ports you did choose for MongoDB and Redis) or at least, if you do, make sure to edit the `docker-compose.yml` properly and the `config.yml` file and add some authentication to both the db and the cache.

MongoDB becomes the source of truth for all the logs that can be retrieved through your node, so make sure the following:
- Do not expose its port publicly without any security mechanism;
- Do not delete the persistent volume where the db is stored if you don't want to sync again;
- Do not manually edit any of the logs inside of the database: this could cause some inconsistency between the real Ethereum Blockchain state and your EthLogSpy instance.

---

## Contributing

We welcome community contributions!

Please check out our <a href="https://github.com/PaoloRollo/ethlogspy/issues">open issues</a> to get started.

If you discover something that could potentially impact security, please notify us immediately by sending an e-mail at <a href="mailto:paolo.rollo1997@gmail.com">paolo.rollo1997@gmail.com</a>. We'll get in touch with you as fast as we can!

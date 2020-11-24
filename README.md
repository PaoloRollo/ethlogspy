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

It exposes the `localhost` node to the world by just reverse proxying every JSON RPC call, except for the `eth_getLogs` one: on this method EthLogSpy retrieves the logs stored in the database (**MongoDB**) or in the cache (**Redis**) and returns it in the same format that the Ethereum Node would use, basically faking its response.

---

## Features

- [x] HTTP JSON RPC API;
- [x] WebSocket JSON RPC API;
- [ ] Support for `eth_subscribe` JSON RPC method. 

---

## Install

TBC.

### Docker

TBC.

### Docker Compose

TBC.

---

## Contributing

We welcome community contributions!

Please check out our <a href="https://github.com/PaoloRollo/ethlogspy/issues">open issues</a> to get started.

If you discover something that could potentially impact security, please notify us immediately by sending an e-mail at <a href="mailto:paolo.rollo1997@gmail.com">paolo.rollo1997@gmail.com</a>. We'll get in touch with you as fast as we can!

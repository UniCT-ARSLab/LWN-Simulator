# LWN Simulator
A LoRaWAN nodes' simulator to simulate a LoRaWAN Network.

## Table of Contents
* [General Info](#general-info)
* [Requirements](#requirements)
* [Installation](#installation)

## General Info
LWN Simulator is a LoRaWAN nodes' simulator equipped with web interface. NS

![dashboard](./readme/dashboard.png)

The project consists of three main components: nodes, forwarder and gateways. 

### The node
* Based [specification LoRAWAN v1.0.3](https://lora-alliance.org/resource_hub/lorawan-specification-v1-0-3/);
* Bear all [LoRaWAN Regional Parameters v1.0.3](https://lora-alliance.org/resource_hub/lorawan-regional-parameters-v1-0-3reva/).
* Implements class A,C and partially even the B class;
* Implements ADR Algorithm;
* Send periodically a frame that including some configurable payload;
* Supports MAC Command;
* Implements FPending procedure;
* It is possibile to interact with it in real-time;

### The forwarder
It receive the frames from devices and create a datagram including them and forward to gateways.

### The gateway
* A virtual gateway configuration specifies that the virtual gateway communicates with the gateway bridge;
* A real gateway configuration specifies that the real gateway istance communicates with the real gateway.

## Requirements
You must install [Go](https://golang.org/ "Go website"). Version >= 1.12.9

## Installation

Firsly, you must clone this repository:
```bash
git clone github.com/...
```
After the download, you must enter in main directory

```bash
cd LWNSimulator
```
and types 

```golang
go run main.go
```
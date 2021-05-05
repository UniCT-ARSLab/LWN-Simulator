# LWN Simulator
A LoRaWAN nodes' simulator to simulate a LoRaWAN Network.

## Table of Contents
* [General Info](#general-info)
* [Requirements](#requirements)
* [Installation](#installation)

## General Info
LWN Simulator is a LoRaWAN nodes' simulator equipped with web interface. It allows to comunicate with a real infrastructure LoRaWAN or ad-hoc infrastructure, such as [Chirpstack](https://www.chirpstack.io/).

![dashboard](./readme/dashboard.png)

The project consists of three main components: devices, forwarder and gateways. 

### The device
* Based [specification LoRAWAN v1.0.3](https://lora-alliance.org/resource_hub/lorawan-specification-v1-0-3/);
* Supports all [LoRaWAN Regional Parameters v1.0.3](https://lora-alliance.org/resource_hub/lorawan-regional-parameters-v1-0-3reva/).
* Implements class A,C and partially even the B class;
* Implements ADR Algorithm;
* Sends periodically a frame that including some configurable payload;
* Supports MAC Command;
* Implements FPending procedure;
* It is possibile to interact with it in real-time;

### The forwarder
It receives the frames from devices, creates a RXPK object including them within and forwards to gateways.

### The gateway
There are two types of gateway:
* A virtual gateway that comunicates with a real gateway bridge (if it exists);
* A real gateway to which datagrams UDP are forwarded.

## Requirements
* If you don't have a real infrastracture, you can download [ChirpStack open-source LoRaWAN® Network Server](https://www.chirpstack.io/project/), or a similar software, to prove it;
* You must install [Go](https://golang.org/ "Go website"). Version >= 1.12.9

## Installation

Firstly, you must clone this repository:
```bash
git clone https://github.com/UniCT-ARSLab/LWN-Simulator.git
```
After the download, you must enter in main directory

```bash
cd LWNSimulator
```
There are two mode to start the simulator:
* In realese mode 

```bash
make run-release
```

* Or ???
```bash
make run
```

After, you open a browser and connect to 127.0.0.1:8000.

Good job :)

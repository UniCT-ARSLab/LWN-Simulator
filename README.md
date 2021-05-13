# LWN Simulator
[![Build Status](https://www.travis-ci.com/UniCT-ARSLab/LWN-Simulator.svg?branch=main)](https://www.travis-ci.com/UniCT-ARSLab/LWN-Simulator)
[![GitHub license](https://img.shields.io/github/license/UniCT-ARSLab/LWN-Simulator)](https://github.com/UniCT-ARSLab/LWN-Simulator/blob/main/LICENSE.txt)
[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](https://golang.org)
[![GitHub go.mod Go version of a Go module](https://img.shields.io/github/go-mod/go-version/UniCT-ARSLab/LWN-Simulator.svg)](https://github.com/UniCT-ARSLab/LWN-Simulator)
[![GitHub release](https://img.shields.io/github/release/UniCT-ARSLab/LWN-Simulator.svg)](https://github.com/UniCT-ARSLab/LWN-Simulator/releases/)

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
* Based [specification LoRaWAN v1.0.3](https://lora-alliance.org/resource_hub/lorawan-specification-v1-0-3/);
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


## Installation

### From binary file
You can download from realeses section the pre-compiled binary file.

[Releases Page](https://github.com/UniCT-ARSLab/LWN-Simulator/releases) 

### From source code

#### Requirements
* You must install [Go](https://golang.org/ "Go website"). Version >= 1.16

Firstly, you must clone this repository:
```bash
git clone https://github.com/UniCT-ARSLab/LWN-Simulator.git
```
After the download, you must enter in main directory:

```bash
cd LWNSimulator
```
You must install all dependencies to build the simulator:
```bash
make install-dep
```
Now you can launch the build of the simulator:
```bash
make build
```

Finally, there are two mode to start the simulator:
* from source (without building the source)
```bash
make run
```
* from the builded binary
```bash
make run-release
```

### Configuration file
The simulator relises on a configuration file (`config.json`) whitch specifies some configurations for the simulator:

```json
{
    "address":"0.0.0.0",
    "port":8000,
    "configDirname":"lwnsimulator"
}
```
* address: specifies the IP mask from which the web UI is accessible.
* port: the web server port.
* configDirname: the directory name where all status files will be saved and will be created in the user home. 

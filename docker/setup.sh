#!/bin/sh

bridgeAddress=${SIM_BRIDGE_ADDRESS:-"172.17.0.1:1700"}
devEUI=${DEV_EUI:-"a000000000000001"}
devAddr=${DEV_ADDR:-"00be1557"}
devPayload=${DEV_PAYLOAD:-"123"}
macAddress=${GW_MAC_ADDRESSE:-"0000000000000002"}

sed -i s/%SIM_BRIDGE_ADDRESS%/$bridgeAddress/g lwnsimulatordata/simulator.json
sed -i s/%DEV_EUI%/$devEUI/g lwnsimulatordata/devices.json
sed -i s/%DEV_ADDR%/$devAddr/g lwnsimulatordata/devices.json
sed -i s/%DEV_PAYLOAD%/$devPayload/g lwnsimulatordata/devices.json
sed -i s/%GW_MAC_ADDRESSE%/$macAddress/g lwnsimulatordata/gateways.json

exec ./lwnsimulator

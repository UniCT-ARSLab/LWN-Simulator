#!/bin/sh
sed -i s/%SIM_BRIDGE_HOST%/$SIM_BRIDGE_HOST/g lwnsimulatordata/simulator.json
sed -i s/%SIM_BRIDGE_PORT%/$SIM_BRIDGE_PORT/g lwnsimulatordata/simulator.json

sed -i s/%GTW_MAC_ADDRESS%/$GTW_MAC_ADDRESS/g lwnsimulatordata/gateways.json
sed -i s/%GTW_NAME%/$GTW_NAME/g lwnsimulatordata/gateways.json

sed -i s/%DEV_EUI%/$DEV_EUI/g lwnsimulatordata/devices.json
sed -i s/%DEV_ADDR%/$DEV_ADDR/g lwnsimulatordata/devices.json
sed -i s/%DEV_NWK_SKEY%/$DEV_NWK_SKEY/g lwnsimulatordata/devices.json
sed -i s/%DEV_APP_SKEY%/$DEV_APP_SKEY/g lwnsimulatordata/devices.json
sed -i s/%DEV_APP_KEY%/$DEV_APP_KEY/g lwnsimulatordata/devices.json
sed -i s/%DEV_NAME%/$DEV_NAME/g lwnsimulatordata/devices.json
sed -i s/%DEV_SENDINTERVAL%/$DEV_SENDINTERVAL/g lwnsimulatordata/devices.json
sed -i s/%DEV_ACKTIMEOUT%/$DEV_ACKTIMEOUT/g lwnsimulatordata/devices.json
sed -i s/%DEV_MYTYPE%/$DEV_MYTYPE/g lwnsimulatordata/devices.json
sed -i s/%DEV_PAYLOAD%/$DEV_PAYLOAD/g lwnsimulatordata/devices.json
sed -i s/%DEV_PAYLOAD_BASE64%/$DEV_PAYLOAD_BASE64/g lwnsimulatordata/devices.json
sed -i s/%DEV_ALIGN_CURRENT_TIME%/$DEV_ALIGN_CURRENT_TIME/g lwnsimulatordata/devices.json
sed -i s/%DEV_INFO_UPLINK_FPORT%/$DEV_INFO_UPLINK_FPORT/g lwnsimulatordata/devices.json
sed -i s/%DEV_CONF_DATARATE%/$DEV_CONF_DATARATE/g lwnsimulatordata/devices.json
sed -i s/%DEV_CONF_RSSI%/$DEV_CONF_RSSI/g lwnsimulatordata/devices.json

sed -i s/%DEV_RXS0_FREQUPLINK%/$DEV_RXS0_FREQUPLINK/g lwnsimulatordata/devices.json
sed -i s/%DEV_RXS0_FREQDOWNLINK%/$DEV_RXS0_FREQDOWNLINK/g lwnsimulatordata/devices.json
sed -i s/%DEV_RXS0_MINDR%/$DEV_RXS0_MINDR/g lwnsimulatordata/devices.json
sed -i s/%DEV_RXS0_MAXDR%/$DEV_RXS0_MAXDR/g lwnsimulatordata/devices.json
sed -i s/%DEV_RXS0_DATARATE%/$DEV_RXS0_DATARATE/g lwnsimulatordata/devices.json

sed -i s/%DEV_RXS1_FREQUPLINK%/$DEV_RXS1_FREQUPLINK/g lwnsimulatordata/devices.json
sed -i s/%DEV_RXS1_FREQDOWNLINK%/$DEV_RXS1_FREQDOWNLINK/g lwnsimulatordata/devices.json
sed -i s/%DEV_RXS1_MINDR%/$DEV_RXS1_MINDR/g lwnsimulatordata/devices.json
sed -i s/%DEV_RXS1_MAXDR%/$DEV_RXS1_MAXDR/g lwnsimulatordata/devices.json
sed -i s/%DEV_RXS1_DATARATE%/$DEV_RXS1_DATARATE/g lwnsimulatordata/devices.json

case $DEV_REGION in
  EU868)
    DEV_REGION=1
    ;;
  US915)
    DEV_REGION=2
    ;;
  CN779)
    DEV_REGION=3
    ;;
  EU433)
    DEV_REGION=4
    ;;
  AU915)
    DEV_REGION=5
    ;;
  CN470)
    DEV_REGION=6
    ;;
  AS923)
    DEV_REGION=7
    ;;
  KR920)
    DEV_REGION=8
    ;;
  IN865)
    DEV_REGION=9
    ;;
  RU864)
    DEV_REGION=10
    ;;
  *)
    DEV_REGION=NULL
    ;;
esac
sed -i s/%DEV_REGION%/$DEV_REGION/g lwnsimulatordata/devices.json

exec ./lwnsimulator
package macCommands

import "github.com/brocaar/lorawan"

//AckMacCommand contains ack command that is need a condition to not send
type AckMacCommand struct {
	rxParamSetupAns  []lorawan.Payload
	dlChannelAns     []lorawan.Payload
	rxTimingSetupAns []lorawan.Payload
}

//SetRXParamSetupAns set ack command for rxParamSetupReq ack
func (c *AckMacCommand) SetRXParamSetupAns(command []lorawan.Payload) {
	c.rxParamSetupAns = append(c.rxParamSetupAns, command...)
}

//GetRXParamSetupAns get RXParamSetupAns command
func (c *AckMacCommand) GetRXParamSetupAns() []lorawan.Payload {
	return c.rxParamSetupAns
}

//SetDLChannelAns set ack command for DLChannelReq ack
func (c *AckMacCommand) SetDLChannelAns(command []lorawan.Payload) {
	c.dlChannelAns = append(c.dlChannelAns, command...)
}

//GetDLChannelAns get DLChannelAns command
func (c *AckMacCommand) GetDLChannelAns() []lorawan.Payload {
	return c.dlChannelAns
}

//SetRXTimingSetupAns set ack command for DLChannelReq ack
func (c *AckMacCommand) SetRXTimingSetupAns(command []lorawan.Payload) {
	c.rxTimingSetupAns = append(c.rxTimingSetupAns, command...)
}

//GetRXTimingSetupAns get DLChannelAns command
func (c *AckMacCommand) GetRXTimingSetupAns() []lorawan.Payload {
	return c.rxTimingSetupAns
}

//CleanFOptsDLChannelAns clean struct
func (c *AckMacCommand) CleanFOptsDLChannelAns() {
	c.dlChannelAns = []lorawan.Payload{}
}

//CleanFOptsRXParamSetupAns clean struct
func (c *AckMacCommand) CleanFOptsRXParamSetupAns() {
	c.rxParamSetupAns = []lorawan.Payload{}
}

//CleanFOptsRXTimingSetupAns clean struct
func (c *AckMacCommand) CleanFOptsRXTimingSetupAns() {
	c.rxTimingSetupAns = []lorawan.Payload{}
}

//GetAll get all ack mac command that require a condition
func (c *AckMacCommand) GetAll() []lorawan.Payload {
	var commands []lorawan.Payload

	ack := c.GetRXParamSetupAns()
	if len(ack) > 0 {
		commands = append(commands, ack...)
	}

	ack = c.GetDLChannelAns()
	if len(ack) > 0 {
		commands = append(commands, ack...)
	}

	ack = c.GetRXTimingSetupAns()
	if len(ack) > 0 {
		commands = append(commands, ack...)
	}

	return commands
}

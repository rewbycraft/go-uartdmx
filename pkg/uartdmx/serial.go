package uartdmx

type UARTDMX interface {
	SetChannel(ch uint16, val uint8)
	SetChannels(startCh uint16, vals []uint8)
	Render() error
	Close() error
}

func MakeUARTDMX(dev string) (UARTDMX, error) {
	return internalMakeUARTDMX(dev)
}

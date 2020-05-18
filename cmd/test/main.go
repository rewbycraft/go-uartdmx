package main

import "github.com/rewbycraft/go-uartdmx/pkg/uartdmx"

func main() {
	thing, err := uartdmx.MakeUARTDMX("/dev/ttyUSB0")
	if err != nil {
		panic(err)
	}

	for i := 0; i < 512; i++ {
		thing.SetChannel(uint16(i), 255)
	}

	for {
		err := thing.Render()
		if err != nil {
			panic(err)
		}
	}

	thing.Close()
}

package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/jsimonetti/go-artnet/packet"
	"github.com/rewbycraft/go-uartdmx/pkg/uartdmx"
)

func artRecvLoop(dmx uartdmx.UARTDMX, pc net.PacketConn, done chan struct{}) {
	keepGoing := true
	buffer := make([]byte, 4096)

	log.Println("Art-Net Receive loop is running.")
	for keepGoing {
		select {
		case <-done:
			keepGoing = false
		default:
			keepGoing = true
		}

		pc.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, _, err := pc.ReadFrom(buffer)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				continue
			} else {
				panic(err)
			}
		}
		p := packet.NewArtDMXPacket()

		err = p.UnmarshalBinary(buffer)
		if err != nil {
			log.Println("Received invalid packet. Ignoring...")
			continue
		}

		dmx.SetChannels(0, p.Data[:p.Length])
	}
	log.Println("Exited recv loop!")
}

func dmxLoop(dmx uartdmx.UARTDMX, done chan struct{}) {
	keepGoing := true
	log.Println("DMX Render loop is running.")
	for keepGoing {
		err := dmx.Render()
		if err != nil {
			panic(err)
		}

		select {
		case <-done:
			keepGoing = false
		default:
			keepGoing = true
		}
	}
	log.Println("Exited dmx loop!")
}

func main() {
	portPtr := flag.String("dev", "/dev/ttyUSB0", "Serial device to use for DMX output.")
	listenPtr := flag.String("listen", ":6454", "Listen string for Art-Net endpoint.")

	flag.Parse()

	log.Println("Opening UART device...")
	dmx, err := uartdmx.MakeUARTDMX(*portPtr)
	if err != nil {
		panic(err)
	}
	defer dmx.Close()

	log.Println("Opening listening port...")
	pc, err := net.ListenPacket("udp", *listenPtr)
	if err != nil {
		return
	}
	defer pc.Close()

	doneArt := make(chan struct{})
	doneDmx := make(chan struct{})
	waiter := sync.WaitGroup{}

	go func() {
		waiter.Add(1)
		artRecvLoop(dmx, pc, doneArt)
		waiter.Done()
	}()

	go func() {
		waiter.Add(1)
		dmxLoop(dmx, doneDmx)
		waiter.Done()
	}()

	log.Println("Waiting for Ctrl-C to exit...")
	WaitForCtrlC()

	doneArt <- struct{}{}
	doneDmx <- struct{}{}

	log.Println("Waiting for threads to exit...")
	waiter.Wait()
}

// https://jjasonclark.com/waiting_for_ctrl_c_in_golang/
func WaitForCtrlC() {
	var end_waiter sync.WaitGroup
	end_waiter.Add(1)
	var signal_channel chan os.Signal
	signal_channel = make(chan os.Signal, 1)
	signal.Notify(signal_channel, os.Interrupt)
	go func() {
		<-signal_channel
		end_waiter.Done()
	}()
	end_waiter.Wait()
}

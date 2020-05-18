package uartdmx

import (
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

type linuxUARTDMX struct {
	dataBuffer []byte
	file       *os.File
}

func (u *linuxUARTDMX) ioctl(request, argp uintptr) error {
	err := syscall.EINTR
	for err == syscall.EINTR {
		_, _, e := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(u.file.Fd()), request, argp, 0, 0, 0)
		err = e
		if err == syscall.EINTR {
			log.Printf("ioctl was interrupted. Retrying...")
		}
	}
	return nil
}

func internalMakeUARTDMX(dev string) (UARTDMX, error) {
	file, openErr :=
		os.OpenFile(
			dev,
			syscall.O_RDWR|syscall.O_NOCTTY,
			0600)
	if openErr != nil {
		return nil, openErr
	}

	s := &linuxUARTDMX{dataBuffer: make([]byte, 512), file: file}

	t := makeTermios2()

	if err := s.ioctl(unix.TCSETS2, uintptr(unsafe.Pointer(t))); err != nil {
		return nil, err
	}

	return s, nil
}

func (u *linuxUARTDMX) sendBreak(enable bool) error {
	c := unix.TIOCSBRK

	if !enable {
		c = unix.TIOCCBRK
	}

	if err := u.ioctl(uintptr(c), uintptr(0)); err != nil {
		return err
	}

	return nil
}

func (u *linuxUARTDMX) flush() error {
	var thing syscall.Termios
	if err := u.ioctl(unix.TCGETS, uintptr(unsafe.Pointer(&thing))); err != nil {
		return fmt.Errorf("get: %w", err)
	}
	if err := u.ioctl(unix.TCSETSW, uintptr(unsafe.Pointer(&thing))); err != nil {
		return fmt.Errorf("set: %w", err)
	}
	return nil
}

func makeTermios2() *unix.Termios {
	t := &unix.Termios{}
	t.Cflag = 0
	t.Cflag |= unix.CSTOPB
	t.Cflag |= unix.CS8
	t.Cflag |= unix.CLOCAL
	t.Cflag |= unix.CREAD
	t.Cflag |= unix.BOTHER
	t.Lflag = 0
	t.Iflag = 0
	t.Oflag = 0
	t.Ispeed = 250000
	t.Ospeed = 250000
	t.Cc[unix.VTIME] = 1
	t.Cc[unix.VMIN] = 0
	return t
}

func (u *linuxUARTDMX) SetChannel(ch uint16, val uint8) {
	u.dataBuffer[ch] = val
}

func (u *linuxUARTDMX) SetChannels(startCh uint16, vals []uint8) {
	copy(u.dataBuffer[startCh:], vals)
	//log.Printf("SET: %03d", u.dataBuffer[0])
}

func (u *linuxUARTDMX) Close() error {
	return u.file.Close()
}

func (u *linuxUARTDMX) Render() error {
	start := time.Now()

	//	log.Printf("DMX: %03d", u.dataBuffer[0])

	if err := u.sendBreak(true); err != nil {
		return fmt.Errorf("Start break: %w", err)
	}

	//Wait for the required break time.
	time.Sleep(110 * time.Microsecond)

	if err := u.sendBreak(false); err != nil {
		return fmt.Errorf("End break: %w", err)
	}

	//Wait for the mark-after-break time.
	time.Sleep(20 * time.Microsecond)

	//Offset somehow? First one's being eaten!
	//TODO: Put the signal analyzer on this and figure out wtf is happening.
	u.file.Write([]byte{0x00})

	n, err := u.file.Write(u.dataBuffer)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if n != 512 {
		return errors.New("Did not send enough bytes!")
	}

	err = u.flush()
	if err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	elapsed := time.Since(start)

	frameTime := int64(33) //One dmx frame is 1s/44Hz = 23ms. 33ms for safety.

	if elapsed.Milliseconds() < frameTime {
		time.Sleep(time.Duration(frameTime-elapsed.Milliseconds()) * time.Millisecond)
	}

	return nil
}

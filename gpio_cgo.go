package gpio

import (
	"fmt"
	"os"
)
/*
#cgo LDFLAGS: -fpic
#include <poll.h>
#include <string.h>
int waitForGpioEvent(int fd) {
	struct pollfd fdset;

	//set the fdset to NULL
	memset((void*)&fdset, 0, sizeof(fdset));

	//set the polling parameters
	fdset.fd = fd;
	fdset.events = POLLPRI;

	//wait for it
	if(poll(&fdset, 1, -1) <= 0) {
		return -1;
	}
	if (fdset.revents & POLLPRI) {
		return 0;
	}
	return -1;
}
*/
import "C"

func (g *GPIO) WaitForFalling() error {
	buff := make([]byte, 8)
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if err := setInterruptEdge(g.id, true); err != nil {
		return err
	}
	//get the file descriptor for the file
	fin, err := os.Open(fmt.Sprintf("/sys/class/gpio/gpio%d/value", g.id))
	if err != nil {
		return err
	}
	defer fin.Close()
	//read to get the first poll to miss
	n, err := fin.Read(buff)
	if err != nil {
		return err
	} else if n != 2 {
		return fmt.Errorf("Failed to perform read on poll")
	}
	if C.waitForGpioEvent(C.int(fin.Fd())) != C.int(0) {
		return fmt.Errorf("Failed to wait on poll")
	}
	return nil
}

func setInterruptEdge(id int, falling bool) error {
	edge := "rising"
	if falling {
		edge = "falling"
	}
	fout, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/edge", id), os.O_RDWR, 0660)
	if err != nil {
		return nil
	}
	defer fout.Close()
	n, err := fmt.Fprintf(fout, "%s\n", edge)
	if err != nil {
		return err
	}
	if n <= 2 {
		return fmt.Errorf("Setting interrupt edge failed")
	}
	return nil
}




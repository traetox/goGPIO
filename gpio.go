package gpio

import (
	"fmt"
	"os"
	"path"
	"sync"
)
/*
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

const (
	baseGpio string = "/sys/class/gpio/"
	export   string = "/sys/class/gpio/export"
)

type GPIO struct {
	id   int
	state bool
	mtx  *sync.Mutex
}

func New(id int) (*GPIO, error) {
	//check if the GPIO has been exported
	if err := checkGpio(id); err != nil {
		if err = exportGpio(id); err != nil {
			return nil, err
		}
		//re check it
		if err = checkGpio(id); err != nil {
			return nil, err
		}
	}
	st, err := getState(id)
	if err != nil {
		return nil, err
	}
	return &GPIO{
		id: id,
		mtx: &sync.Mutex{},
		state: st,
	}, nil
}

func (g *GPIO) SetInput() error {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if err := setDirection(g.id, true); err != nil {
		return err
	}
	return nil
}

func (g *GPIO) SetOutput() error {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if err := setDirection(g.id, false); err != nil {
		return err
	}
	return nil
}

func (g *GPIO) Toggle() error {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if err := setState(g.id, g.state); err != nil {
		return err
	}
	g.state = !g.state
	return nil
}

func (g *GPIO) Off() error {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if err := setState(g.id, false); err != nil {
		return err
	}
	g.state = false
	return nil
}

func (g *GPIO) On() error {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if err := setState(g.id, true); err != nil {
		return err
	}
	g.state = true
	return nil
}

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


func checkGpio(id int) error {
	fi, err := os.Stat(path.Join(baseGpio, fmt.Sprintf("gpio%d", id)))
	if err != nil {
		return err
	}
	//check that it is a dir
	if !fi.Mode().IsDir() {
		return fmt.Errorf("%d is not exported", id)
	}
	return nil
}

func exportGpio(id int) error {
	fout, err := os.OpenFile(export, os.O_WRONLY, 0660)
	if err != nil {
		return nil
	}
	defer fout.Close()
	nn, err := fout.Seek(0, 0)
	if err != nil {
		return err
	}
	if nn != 0 {
		return fmt.Errorf("Failed to seek")
	}
	n, err := fmt.Fprintf(fout, "%d\n", id)
	if err != nil {
		return err
	}
	if n < 2 {
		return fmt.Errorf("Failed to export %d", n)
	}
	return nil
}

func setDirection(id int, in bool) error {
	direction := "out"
	if in {
		direction = "in"
	}
	fout, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/direction", id), os.O_RDWR, 0660)
	if err != nil {
		return nil
	}
	defer fout.Close()
	n, err := fmt.Fprintf(fout, "%s\n", direction)
	if err != nil {
		return err
	}
	if n <= 2 {
		return fmt.Errorf("Setting direction failed")
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


func getState(id int) (bool, error) {
	buff := make([]byte, 16)
	fin, err := os.Open(fmt.Sprintf("/sys/class/gpio/gpio%d/value", id))
	if err != nil {
		return false, err
	}
	defer fin.Close()
	n, err := fin.Read(buff)
	if err != nil {
		return false, err
	}
	if n != 2 {
		return false, fmt.Errorf("Invalid response: %s\n", string(buff[0:n]))
	}
	if buff[0] == '1' {
		return true, nil
	} else if buff[0] == '0' {
		return false, nil
	}
	return false, fmt.Errorf("Invalid response: %s\n", string(buff[0:n]))
}

func setState(id int, on bool) error {
	value := "0"
	if on {
		value = "1"
	}
	fout, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/value", id), os.O_RDWR, 0660)
	if err != nil {
		return err
	}
	defer fout.Close()
	n, err := fmt.Fprintf(fout, "%s\n", value)
	if err != nil {
		return err
	}
	if n != 2 {
		return fmt.Errorf("Failed to set value")
	}
	return nil
}

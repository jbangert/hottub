package controller

import (
	"bufio"
	"github.com/tarm/serial"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Hottub struct {
	targetTemp float64

	inletTemp     float64
	outletTemp    float64
	statusMessage string

	heater chan bool

	mu sync.Mutex
}

func (h *Hottub) Start() {
	// TODO(bangert): Start socket
	log.Printf("Starting hottub process")
	config := &serial.Config{Name: "/dev/ttyAMA0", Baud: 57600}
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("Cannot open serial port")
	}
	h.heater = make(chan bool)
	go h.communicateCommand(port)
	go h.communicateSensor(port)
	go h.control()
	if err != nil {
		log.Printf("Error when running screen %v", err)
	}
}

func (h *Hottub) control() {
	for {
		time.Sleep(1000 * time.Millisecond)
		if h.GetInletTemp() < h.GetTargetTemp() - 1 {
			h.heater <- true
		} else if h.GetInletTemp() > h.GetTargetTemp() {
			h.heater <- false
		}
	}
}

func (h *Hottub) communicateCommand(arduino io.WriteCloser) {
	heaterCommand := false
	for {
		select {
		case heaterCommand = <-h.heater:
		case <-time.After(500 * time.Millisecond):
		}
		var output [1]byte
		if heaterCommand {
			output[0] = byte('+')
		} else {
			output[0] = byte('-')
		}
		_, err := arduino.Write(output[:])
		if err != nil {
			log.Printf("Error writing to arduino %v", err)
		}
	}
}

func (h *Hottub) communicateSensor(rawArduino io.ReadCloser) {
	arduino := bufio.NewReader(rawArduino)
	for {
		line, err := arduino.ReadString('\n')
		if err != nil {
			log.Fatalf("cannot read input %v", err)
		}
		parsed := strings.Split(line, ":")
		if len(parsed) != 2 {
			log.Printf("Invalid line received from Arduino %v", line)
			continue
		}
		field, value := parsed[0], parsed[1]
		value = strings.TrimSuffix(value, "\r\n")
		if value == "85.00C" || value == "-127.00C" {
			continue
		}

		switch field {
		case "28FF31DC7016584":
			h.mu.Lock()
			h.inletTemp, err = strconv.ParseFloat(strings.TrimSuffix(value, "C"), 64)
			h.mu.Unlock()
			if err != nil {
				log.Printf("Cannot parse float %v", err)
				continue
			}
		case "28FF1D647116425":
			h.mu.Lock()
			h.outletTemp, err = strconv.ParseFloat(strings.TrimSuffix(value, "C"), 64)
			h.mu.Unlock()
			if err != nil {
				log.Printf("Cannot parse float %v", err)
				continue
			}
		case "Status":
			// Actual status
			h.mu.Lock()
			h.statusMessage = value
			h.mu.Unlock()
		default:
			log.Printf("Unknown field %v with value %v received", field, value)
		}

	}
}

func (h *Hottub) GetTargetTemp() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.targetTemp
}
func (h *Hottub) SetTargetTemp(temp float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.targetTemp = temp
}

func (h *Hottub) GetInletTemp() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.inletTemp
}

func (h *Hottub) GetOutletTemp() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.outletTemp
}

func (h *Hottub) GetStatus() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.statusMessage
}

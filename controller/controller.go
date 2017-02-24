package controller

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Hottub struct {
	targetTemp float64

	inletTemp     float64
	outletTemp    float64
	statusMessage string

	heater chan bool
}

func (h *Hottub) Run() {
	// TODO(bangert): Start socket
	log.Printf("Starting hottub process")
	cmd := exec.Command("screen -S hottub /dev/ttyAMA0 57600")
	h.heater = make(chan bool)
	inpipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Cannot open pipe %v", err)
	}
	go h.communicateSensor(inpipe)
	//	outpipe, err := cmd.StdoutPipe()
	//	if err != nil {
	//		log.Fatalf("Cannot open output pipe %v", err)

	err = cmd.Run()

	if err != nil {
		log.Printf("Error when running screen %v", err)
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
		if value == "85.00C" || value == "-127.00C" {
			continue
		}

		switch field {
		case "28FF31DC7016584":
			h.inletTemp, err = strconv.ParseFloat(strings.TrimSuffix(value, "C"), 64)
			if err != nil {
				log.Fatalf("Cannot parse float %v", err)
			}
		case "28FF1D647116425":
			h.outletTemp, err = strconv.ParseFloat(strings.TrimSuffix(value, "C"), 64)
			if err != nil {
				log.Fatalf("Cannot parse float %v", err)
			}
		case "Status":
			// Actual status
			h.statusMessage = value
		default:
			log.Printf("Unknown field %v with value %v received", field, value)
		}

	}
}

func (h *Hottub) GetTargetTemp() float64 {
	return h.targetTemp
}
func (h *Hottub) SetTargetTemp(temp float64) {
	h.targetTemp = temp
}

func (h *Hottub) GetInletTemp() float64 {
	return h.inletTemp
}

func (h *Hottub) GetOutletTemp() float64 {
	return h.outletTemp
}

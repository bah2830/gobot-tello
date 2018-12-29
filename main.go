package main

import (
	"fmt"
	"math"
	"os/exec"
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/dji/tello"
	"gobot.io/x/gobot/platforms/joystick"
)

const (
	videoBitrate = tello.VideoBitRate4M
)

var (
	xboxLimits = Axis{
		LeftY:  AxisLimit{Min: -32768, Max: 32767},
		LeftX:  AxisLimit{Min: -32768, Max: 32767},
		RightY: AxisLimit{Min: -32768, Max: 32767},
		RightX: AxisLimit{Min: -32768, Max: 32767},
	}

	pressed       = AxisPressed{}
	currentStatus = &CurrentStatus{}
)

type Axis struct {
	LeftY  AxisLimit
	LeftX  AxisLimit
	RightY AxisLimit
	RightX AxisLimit
}
type AxisLimit struct {
	Min int16
	Max int16
}

type AxisPressed struct {
	LeftX  bool
	LeftY  bool
	RightX bool
	RightY bool
}

type ButtonToggled struct {
	Y bool
}

type CurrentStatus struct {
	Flying         bool
	FastMode       bool
	BatteryPercent int8
	WifiSignal     int8
	Status         string
}

func main() {
	controllerAdapter := joystick.NewAdaptor()
	stick := joystick.NewDriver(controllerAdapter, joystick.Xbox360)

	drone := tello.NewDriver("8888")

	work := func() {
		mpv := exec.Command("mplayer", "-fps", "60", "-")
		mpvIn, _ := mpv.StdinPipe()
		if err := mpv.Start(); err != nil {
			fmt.Println(err)
			return
		}

		drone.On(tello.ConnectedEvent, func(data interface{}) {
			fmt.Println("Connected")
			drone.StartVideo()
			drone.SetVideoEncoderRate(videoBitrate)
			gobot.Every(100*time.Millisecond, func() {
				drone.StartVideo()
			})
		})
		drone.On(tello.VideoFrameEvent, func(data interface{}) {
			pkt := data.([]byte)
			if _, err := mpvIn.Write(pkt); err != nil {
				fmt.Println(err)
			}
		})
		drone.On(tello.FlightDataEvent, func(data interface{}) {
			flightData := data.(*tello.FlightData)
			currentStatus.SetData(flightData)
		})

		// Setup controller controls
		stick.On(joystick.StartPress, func(data interface{}) {
			if currentStatus.Flying {
				drone.Land()
				currentStatus.SetStatus("landing")
				currentStatus.Flying = false
			} else {
				drone.TakeOff()
				currentStatus.SetStatus("take off")
				currentStatus.Flying = true
			}
		})

		stick.On(joystick.RBPress, func(data interface{}) {
			if currentStatus.FastMode {
				currentStatus.ModeSlow()
				drone.SetSlowMode()

			} else {
				currentStatus.ModeFast()
				drone.SetFastMode()
			}
		})

		stick.On(joystick.APress, func(data interface{}) {
			drone.BackFlip()
		})

		stick.On(joystick.YPress, func(data interface{}) {
			drone.FrontFlip()
		})

		stick.On(joystick.XPress, func(data interface{}) {
			drone.LeftFlip()
		})

		stick.On(joystick.BPress, func(data interface{}) {
			drone.RightFlip()
		})

		stick.On(joystick.RightX, func(data interface{}) {
			input := data.(int16)
			percent := xboxLimits.RightX.ToPercent(input)

			if input == 0 {
				if pressed.RightX {
					drone.Right(0)
				}

				pressed.RightX = false
			} else {
				pressed.RightX = true
				if input > 0 {
					drone.Right(percent)
				} else {
					drone.Left(percent)
				}
			}
		})

		stick.On(joystick.RightY, func(data interface{}) {
			input := data.(int16)
			percent := xboxLimits.RightY.ToPercent(input)

			if input == 0 {
				if pressed.RightY {
					drone.Forward(0)
				}

				pressed.RightY = false
			} else {
				pressed.RightY = true
				if input > 0 {
					drone.Backward(percent)
				} else {
					drone.Forward(percent)
				}
			}
		})

		stick.On(joystick.LeftX, func(data interface{}) {
			input := data.(int16)
			percent := xboxLimits.LeftX.ToPercent(input)

			if input == 0 {
				if pressed.LeftX {
					drone.Clockwise(0)
				}

				pressed.LeftX = false
			} else {
				pressed.LeftX = true
				if input > 0 {
					drone.Clockwise(percent)
				} else {
					drone.CounterClockwise(percent)
				}
			}
		})

		stick.On(joystick.LeftY, func(data interface{}) {
			input := data.(int16)
			percent := xboxLimits.LeftY.ToPercent(input)

			if input == 0 {
				if pressed.LeftY {
					drone.Up(0)
				}

				pressed.LeftY = false
			} else {
				pressed.LeftY = true
				if input > 0 {
					drone.Down(percent)
				} else {
					drone.Up(percent)
				}
			}
		})
	}

	robot := gobot.NewRobot("tello",
		[]gobot.Connection{controllerAdapter},
		[]gobot.Device{drone, stick},
		work,
	)

	robot.Start()
}

func (a AxisLimit) ToPercent(input int16) int {
	if input == 0 {
		return 0
	}

	var percent int
	if input > 0 {
		percent = int((float64(input) / float64(a.Max)) * 100.0)
	} else {
		percent = int((math.Abs(float64(input)) / math.Abs(float64(a.Min))) * 100.0)
	}

	// Handle drift on the controller
	if percent <= 5 {
		return 0
	}

	return percent
}

func (s *CurrentStatus) SetData(data *tello.FlightData) {
	s.BatteryPercent = data.BatteryPercentage
	s.WifiSignal = data.WifiStrength

	s.Print()
}

func (s *CurrentStatus) ModeFast() {
	s.FastMode = true
	s.Print()
}

func (s *CurrentStatus) ModeSlow() {
	s.FastMode = false
	s.Print()
}

func (s *CurrentStatus) SetStatus(status string) {
	s.Status = status
	s.Print()
}

func (s *CurrentStatus) Print() {
	mode := "slow"
	if s.FastMode {
		mode = "fast"
	}

	fmt.Printf(
		"\rStatus: %-20vMode: %-15vBattery: %-10vWifi: %-10v",
		s.Status,
		mode,
		s.BatteryPercent,
		s.WifiSignal,
	)
}

package main

import (
	"fmt"
	"math"
	"os/exec"
	"time"

	"gobot.io/x/gobot/platforms/joystick"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/dji/tello"
)

const (
	videoBitrate = tello.VideoBitRate15M
)

var (
	xboxLimits = Axis{
		LeftY:  AxisLimit{Min: -32768, Max: 32767},
		LeftX:  AxisLimit{Min: -32768, Max: 32767},
		RightY: AxisLimit{Min: -32768, Max: 32767},
		RightX: AxisLimit{Min: -32768, Max: 32767},
	}

	pressed = AxisPressed{}
	toggled = ButtonToggled{}
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

func main() {
	controllerAdapter := joystick.NewAdaptor()
	stick := joystick.NewDriver(controllerAdapter, joystick.Xbox360)

	drone := tello.NewDriver("8888")

	work := func() {
		mpv := exec.Command("mpv", "--fps", "60", "-")
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
			printData(flightData)
		})

		// Setup controller controls
		stick.On(joystick.APress, func(data interface{}) {
			drone.TakeOff()
		})
		stick.On(joystick.BPress, func(data interface{}) {
			drone.Land()
		})
		stick.On(joystick.YPress, func(data interface{}) {
			if toggled.Y {
				toggled.Y = false
				drone.SetSlowMode()
				fmt.Println("Slow mode enabled")
			} else {
				toggled.Y = true
				drone.SetFastMode()
				fmt.Println("Fast mode enabled")
			}
		})
		stick.On(joystick.L1Press, func(data interface{}) {
			drone.BackFlip()
		})
		stick.On(joystick.R1Press, func(data interface{}) {
			drone.FrontFlip()
		})
		stick.On(joystick.L2Press, func(data interface{}) {
			drone.LeftFlip()
		})
		stick.On(joystick.R2Press, func(data interface{}) {
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

func printData(data *tello.FlightData) {
	fmt.Printf(
		"\rFast Mode: %t\t\tBattery: %d%%\t\tWifi: %d",
		toggled.Y,
		data.BatteryPercentage,
		data.WifiStrength,
	)
}

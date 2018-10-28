package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"net"
	"strconv"

	"github.com/robfig/cron"
)

type Device struct {
	Name string
	IP string
	Role string
	Status string
	ConnectedAt string
}

func redirectDashboard(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/dashboard/", 301)
}

func runCmd(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).Output()
	_ = err

	return string(out)
}

func getAvailablePort(start, max int) int {
	for i := start; i < start + max; i++ {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", strconv.Itoa(i)), 750 * time.Millisecond)

		_ = err

		if conn != nil {
			conn.Close()
			return i
		}
	}

	return -1
}

func main() {
	var devices []Device

	c := cron.New()

	counter := 0

	c.AddFunc("0 0 * * * *", func() {
		currentTime := time.Now()

		cmd := "echo " + currentTime.Format("2006.01.02-15:04:05") + "#" + strings.Split(runCmd("cat /sys/devices/virtual/thermal/thermal_zone0/temp"), "\n")[0] + " >> ./data/temperature.data"

		runCmd("tail -n 48 data/temperature.data > data/temperature.tmp && mv data/temperature.tmp data/temperature.data")
		runCmd(cmd)

		counter += 1
	})

	c.Start()

	//DASHBOARD AT /dashboard
	http.Handle("/dashboard/", http.StripPrefix("/dashboard/", http.FileServer(http.Dir("template"))))

	//CONTROL AT /control
	http.HandleFunc("/control/", func(w http.ResponseWriter, r *http.Request) {
		command := strings.Join(strings.Split(r.URL.Path[1:], "/")[1:], "/")

		category := strings.Split(command, "/")[0]

		var arg []string

		if len(strings.Split(command, "/")) > 1 {
			arg = strings.Split(command, "/")[1:]
		}

		if category == "info" {
			switch arg[0] {
			case "temperature":
				if len(arg) > 1 {
					if arg[1] == "all" {
						w.Write([]byte(runCmd("cat ./data/temperature.data")))
					}
				} else {
					w.Write([]byte(runCmd("cat /sys/devices/virtual/thermal/thermal_zone0/temp")))
				}
			case "uptime":
				w.Write([]byte(strings.Split(runCmd("cat /proc/uptime"), ".")[0]))
			}
		} else if category == "led" {
			var cmd string

			switch arg[0] {
			case "on":
				cmd = "echo 1 > /sys/class/leds/red_led/brightness"

			case "off":
				cmd = "echo 0 > /sys/class/leds/red_led/brightness"
			}

			output := runCmd(cmd)

			if len(output) > 0 {
				w.Write([]byte("\n\nOutput: " + output))
				fmt.Println(output)
			}
		} else if category == "devices" {
			switch arg[0] {
			case "new":
				if len(arg) >= 4 {
					ip := arg[1]
					name := arg[2]
					role := arg[3]

					devices = append(devices, Device{Name: name, IP: ip, Role: role, Status: "UP", ConnectedAt: time.Now().Format("2006.01.02-15:04:05")})

					w.Write([]byte(strconv.Itoa(getAvailablePort(51000, 256))))
				} else {
					w.Write([]byte("ERROR - Not enough arguments."))
				}
			case "all":
				if len(arg) == 1 {
					for _, element := range devices {
						w.Write([]byte(element.Name + "\n"))
						w.Write([]byte(element.IP + "\n"))
						w.Write([]byte(element.Role + "\n"))
						w.Write([]byte(element.Status + "\n"))
						w.Write([]byte(element.ConnectedAt + "\n"))
						w.Write([]byte("\n"))
					}
				} else if len(arg) >= 2 && arg[1] == "json" {
					for _, element := range devices {
						if element != (Device{"", "", "", "", ""}) {
							w.Write([]byte("{ \"name\": \""))
							w.Write([]byte(element.Name + "\", \"ip\": \""))
							w.Write([]byte(element.IP + "\", \"role\": \""))
							w.Write([]byte(element.Role + "\", \"status\": \""))
							w.Write([]byte(element.Status + "\", \"connectedat\": \""))
							w.Write([]byte(element.ConnectedAt + "\" }\n"))
						}
					}
				}
			}
		}
	})

	http.HandleFunc("/", redirectDashboard)

	runCmd("kill -9 " + os.Args[2])
	runCmd("echo 1 > /sys/class/leds/green_led/brightness")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
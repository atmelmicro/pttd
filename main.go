package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/grafov/evdev"
)

func runAsUser(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)

	cmd.Env = append(os.Environ(),
		"XDG_RUNTIME_DIR=/run/user/1000",
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus",
	)

	return cmd
}

func main() {
	var verbose bool
	var keycode string
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.StringVar(&keycode, "k", "KEY_LEFTMETA", "keycode to listen for (Linux input event code)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [-v] [-k keycode] /dev/input/by-id/<device-name>\n", os.Args[0])
		os.Exit(1)
	}

	devicePath := flag.Arg(0)

	dev, err := evdev.Open(devicePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open device: %v\n", err)
		if os.Getuid() != 0 {
			fmt.Fprintf(os.Stderr, "Fix permissions to %s or run as root\n", devicePath)
		}
		os.Exit(1)
	}
	defer dev.File.Close()

	fmt.Fprintf(os.Stderr, "Input device name: %q\n", dev.Name)
	fmt.Fprintf(os.Stderr, "Input device ID: bus %#x vendor %#x product %#x\n",
		dev.Bustype, dev.Vendor, dev.Product)

	evKeycode := evdev.Ecode(keycode)
	if evKeycode == 0 && keycode != "KEY_RESERVED" {
		fmt.Fprintf(os.Stderr, "Key code not found\n")
		fmt.Fprintf(os.Stderr, "see https://github.com/torvalds/linux/blob/master/include/uapi/linux/input-event-codes.h\n")
		os.Exit(1)
	}

	keyCaps, ok := dev.Capabilities["EV_KEY"]
	if !ok {
		fmt.Fprintf(os.Stderr, "This device is not capable of sending this key code\n")
		os.Exit(1)
	}
	if _, ok := keyCaps[evKeycode]; !ok {
		fmt.Fprintf(os.Stderr, "This device is not capable of sending this key code\n")
		os.Exit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Listening for code %s\n", keycode)
	}

	for {
		events, err := dev.Read()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
			continue
		}

		for _, ev := range events {
			if ev.Type == evdev.EV_KEY && ev.Code == uint16(evKeycode) && ev.Value != 2 {
				if verbose {
					if ev.Value == 1 {
						fmt.Fprintf(os.Stderr, "key down\n")
						err := runAsUser("wpctl", "set-mute", "@DEFAULT_AUDIO_SOURCE@", "0").Run()
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error toggling mute: %v\n", err)
						}
					} else {
						fmt.Fprintf(os.Stderr, "key up\n")
						err := runAsUser("wpctl", "set-mute", "@DEFAULT_AUDIO_SOURCE@", "1").Run()
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error toggling mute: %v\n", err)
						}
					}
				}
			}
		}
	}
}

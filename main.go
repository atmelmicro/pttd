package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/gogpu/systray"
	"github.com/grafov/evdev"
)

type Icon struct {
	dark  []byte
	light []byte
}

type Icons struct {
	micDisabledIcon Icon
	micOffIcon      Icon
	micOnIcon       Icon
}

type Settings struct {
	delay   int
	device  string
	keycode string
	verbose bool
}

func getSettings() Settings {
	configDir, err := os.UserConfigDir()
	if err != nil {
		panic("cant get user config dir, make a github issue if you see this")
	}

	configPath := filepath.Join(configDir, "pttd", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		panic("could not read config file, make you created the config file at " + configPath)
	}
	dataStr := string(data)
	var settings Settings
	if err := yaml.Unmarshal([]byte(dataStr), &settings); err != nil {
		panic("could not parse config file, make sure it is valid yaml")
	}
	return settings
}

func setIcon(tray *systray.SystemTray, icon Icon) {
	tray.SetIcon(icon.light).SetDarkModeIcon(icon.dark)
}

func loadIcon(path string) []byte {
	icon, err := os.ReadFile("icons/" + path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading icon %s: %v\n", path, err)
		os.Exit(1)
	}

	return icon
}

func loadIcons() Icons {
	micDisabledIconDark := loadIcon("dark/mic-disabled.png")
	micDisabledIconLight := loadIcon("light/mic-disabled.png")

	micOffIconDark := loadIcon("dark/mic-off.png")
	micOffIconLight := loadIcon("light/mic-off.png")

	micOnIconDark := loadIcon("dark/mic-on.png")
	micOnIconLight := loadIcon("light/mic-on.png")

	return Icons{
		micDisabledIcon: Icon{dark: micDisabledIconDark, light: micDisabledIconLight},
		micOffIcon:      Icon{dark: micOffIconDark, light: micOffIconLight},
		micOnIcon:       Icon{dark: micOnIconDark, light: micOnIconLight},
	}
}

func setMute(value string) {
	err := exec.Command("wpctl", "set-mute", "@DEFAULT_AUDIO_SOURCE@", value).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error toggling mute: %v\n", err)
	}
}

func createMenu(isEnabled *bool, tray *systray.SystemTray, icons Icons) *systray.Menu {
	menu := systray.NewMenu()

	menu.AddCheckbox("Enabled", *isEnabled, func() {
		*isEnabled = !*isEnabled

		// If the user is disabeling the mute, make sure to unmute the mic first
		var wpctlArg string
		var menuIcon Icon
		if *isEnabled {
			wpctlArg = "1"
			menuIcon = icons.micOffIcon
		} else {
			wpctlArg = "0"
			menuIcon = icons.micDisabledIcon
		}

		setMute(wpctlArg)

		newMenu := createMenu(isEnabled, tray, icons)
		fmt.Println("setting new menu", isEnabled)
		setIcon(tray, menuIcon)
		tray.SetMenu(newMenu)
	})

	menu.Add("Quit", func() {
		tray.Remove()
		os.Exit(0)
	})

	return menu
}

func runTray(tray *systray.SystemTray) {
	if err := tray.Run(); err != nil {
		fmt.Println("error:", err)
	}
}

func main() {
	settings := getSettings()
	// mute mic on startup

	setMute("1")
	// Load icons
	icons := loadIcons()

	// Create system tray
	tray := systray.New()
	setIcon(tray, icons.micOffIcon)

	isEnabled := true
	menu := createMenu(&isEnabled, tray, icons)
	tray.SetMenu(menu)
	tray.Show()

	go runTray(tray)

	devicePath := settings.device

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

	evKeycode := evdev.Ecode(settings.keycode)
	if evKeycode == 0 && settings.keycode != "KEY_RESERVED" {
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

	if settings.verbose {
		fmt.Fprintf(os.Stderr, "Listening for code %s\n", settings.keycode)
	}

	for {
		events, err := dev.Read()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
			continue
		}

		for _, ev := range events {
			if ev.Type == evdev.EV_KEY && ev.Code == uint16(evKeycode) && ev.Value != 2 {
				if settings.verbose {
					fmt.Fprintf(os.Stderr, "Received event: type=%d code=%d value=%d\n", ev.Type, ev.Code, ev.Value)
				}

				if ev.Value == 1 {
					fmt.Fprintf(os.Stderr, "key down\n")
					setMute("0")
					setIcon(tray, icons.micOnIcon)
				} else {
					fmt.Fprintf(os.Stderr, "key up\n")
					setMute("1")
					setIcon(tray, icons.micOffIcon)
				}
			}
		}
	}
}

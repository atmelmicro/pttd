package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/gogpu/systray"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/grafov/evdev"
)

//go:embed assets/**
var assetsFS embed.FS

type Icon struct {
	kde struct {
		dark  []byte
		light []byte
	}
	gnome struct {
		dark  []byte
		light []byte
	}
}

type Icons struct {
	micDisabledIcon Icon
	micOffIcon      Icon
	micOnIcon       Icon
}

type Settings struct {
	Delay   int     `yaml:"delay"`
	Device  string  `yaml:"device"`
	Keycode string  `yaml:"keycode"`
	Verbose bool    `yaml:"verbose"`
	Volume  float64 `yaml:"volume"`
	Muted   bool    `yaml:"muted"`
	Kde     bool    `yaml:"kde"`
}

type Sound struct {
	Buffer *beep.Buffer
}

// sound

func loadOgg(path string) *Sound {
	f, err := assetsFS.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	streamer, format, err := vorbis.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)

	return &Sound{Buffer: buffer}
}

func play(s *Sound, volume float64) {
	stream := s.Buffer.Streamer(0, s.Buffer.Len())

	vol := &effects.Volume{
		Streamer: stream,
		Base:     2,
		Volume:   volume,
		Silent:   false,
	}

	speaker.Play(vol)
}

// settings

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

// tray

func setIcon(tray *systray.SystemTray, icon Icon, settings Settings) {
	// Check if we are running kde
	if settings.Kde {

		tray.SetIcon(icon.kde.light).SetDarkModeIcon(icon.kde.dark)
	} else {
		tray.SetIcon(icon.gnome.light).SetDarkModeIcon(icon.gnome.dark)
	}
}

func loadIcon(path string) []byte {
	icon, err := assetsFS.ReadFile("assets/icons/" + path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading icon %s: %v\n", path, err)
		os.Exit(1)
	}

	return icon
}

func loadIconSet(path string) Icon {
	gnomeDarkIcon := loadIcon("gnome/dark/" + path)
	gnomeLightIcon := loadIcon("gnome/light/" + path)

	kdeDarkIcon := loadIcon("kde/dark/" + path)
	kdeLightIcon := loadIcon("kde/light/" + path)

	return Icon{
		kde: struct {
			dark  []byte
			light []byte
		}{dark: kdeDarkIcon, light: kdeLightIcon},
		gnome: struct {
			dark  []byte
			light []byte
		}{dark: gnomeDarkIcon, light: gnomeLightIcon},
	}
}

func loadIcons() Icons {
	return Icons{
		micDisabledIcon: loadIconSet("mic-disabled.png"),
		micOffIcon:      loadIconSet("mic-off.png"),
		micOnIcon:       loadIconSet("mic-on.png"),
	}
}

// recreating the menu is stupid, but i dont think this lib supports doing this any other way
func createMenu(isEnabled *bool, tray *systray.SystemTray, icons Icons, settings Settings) *systray.Menu {
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

		newMenu := createMenu(isEnabled, tray, icons, settings)
		setIcon(tray, menuIcon, settings)
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

// mic control

func setMute(value string) {
	err := exec.Command("wpctl", "set-mute", "@DEFAULT_AUDIO_SOURCE@", value).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error toggling mute: %v\n", err)
	}
}

func main() {
	settings := getSettings()

	// load sounds
	format := beep.SampleRate(44100)
	speaker.Init(format, format.N(time.Second/100))

	in := loadOgg("assets/sounds/in.ogg")
	out := loadOgg("assets/sounds/out.ogg")

	// mute mic on startup

	setMute("1")
	icons := loadIcons()

	tray := systray.New()
	setIcon(tray, icons.micOffIcon, settings)

	isEnabled := true
	menu := createMenu(&isEnabled, tray, icons, settings)
	tray.SetMenu(menu)
	tray.Show()

	go runTray(tray)

	devicePath := settings.Device

	dev, err := evdev.Open(devicePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open device: %v\n", err)
		if os.Getuid() != 0 {
			fmt.Fprintf(os.Stderr, "Fix permissions to %s \n", devicePath)
		}
		os.Exit(1)
	}
	defer dev.File.Close()

	fmt.Fprintf(os.Stderr, "Input device name: %q\n", dev.Name)
	fmt.Fprintf(os.Stderr, "Input device ID: bus %#x vendor %#x product %#x\n",
		dev.Bustype, dev.Vendor, dev.Product)

	evKeycode := evdev.Ecode(settings.Keycode)
	if evKeycode == 0 && settings.Keycode != "KEY_RESERVED" {
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

	if settings.Verbose {
		fmt.Fprintf(os.Stderr, "Listening for code %s\n", settings.Keycode)
	}

	var timer *time.Timer = nil

	for {
		events, err := dev.Read()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
			// probably the device was disconnected, wait a bit before trying again
			time.Sleep(5000 * time.Millisecond)
			continue
		}

		for _, ev := range events {
			if ev.Type == evdev.EV_KEY && ev.Code == uint16(evKeycode) && ev.Value != 2 {
				if !isEnabled {
					continue
				}
				if settings.Verbose {
					fmt.Fprintf(os.Stderr, "Received event: type=%d code=%d value=%d\n", ev.Type, ev.Code, ev.Value)
				}

				if ev.Value == 1 {
					if timer != nil {
						timer.Stop()
						timer = nil
					}

					setMute("0")
					setIcon(tray, icons.micOnIcon, settings)

					if !settings.Muted {
						play(in, settings.Volume)
					}
				} else {
					timer = time.AfterFunc(time.Duration(settings.Delay)*time.Millisecond, func() {
						setMute("1")
						setIcon(tray, icons.micOffIcon, settings)

						if !settings.Muted {
							play(out, settings.Volume)
						}
					})
				}
			}
		}
	}
}

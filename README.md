# pttd (Push-to-talk daemon)

**pttd** is a simple app that fixes push-to-talk not working when the window is not in focus. This is most needed in apps like Discord or TeamSpeak, but pttd is system level, so it works with any app.

## Installation

You can install **pttd** on any Linux machine that is running PipeWire. This includes pretty much any modern desktop Linux distro. Immutable distributions are fully supported.

### Step 1 - Add yourself to the `input` group

On Bazzite

- Go to the Bazzite Portal
- Search for "input"
- Select "Add \<\<input>> to your user groups"
- Then select "Add User to \<\<input>> Group"
- Reboot

For the rest of the Linux distributions

- Open a terminal
- Run

```bash
sudo usermod -aG input $USER
```

- Reboot

### Step 2 - Figure out your device-id

- Open a terminal
- Run

```bash
evtest
```

- You should see a list of all your input devices
  - If you do not see your input device, please create a GitHub issue
- For the device you want to use, take note of all the event numbers
- In my case it will be my mouse, so all the entries are

````

/dev/input/event10: Compx 2.4G Wireless 8K Receiver
/dev/input/event11: Compx 2.4G Wireless 8K Receiver
/dev/input/event12: Compx 2.4G Wireless 8K Receiver
/dev/input/event13: Compx 2.4G Wireless 8K Receiver Consumer Control
/dev/input/event14: Compx 2.4G Wireless 8K Receiver System Control
/dev/input/event15: Compx 2.4G Wireless 8K Receiver
/dev/input/event16: Compx 2.4G Wireless 8K Receiver
/dev/input/event17: Compx 2.4G Wireless 8K Receiver Mouse
/dev/input/event18: Compx 2.4G Wireless 8K Receiver

```

- Now type the first event number (for `/dev/input/event10` -> 10) and press Enter
- Try to press the button you are trying to bind to your PTT key
  - If nothing happens, press `Ctrl+C` to exit and try the next one. Do this for all the events
  - If you are not able to get your event + keycode, create a GitHub issue
- For me, `18` was the correct one. When I press my side button on my mouse, I see this in the terminal
- You will see something like this

```

Event: time 1781641240.046489, -------------- SYN_REPORT ------------
Event: time 1781641240.145495, type 4 (EV_MSC), code 4 (MSC_SCAN), value 90005
Event: time 1781641240.145495, type 1 (EV_KEY), code 276 (BTN_EXTRA), value 0
Event: time 1781641240.145495, -------------- SYN_REPORT ------------

```

- The important part is "Event: time 1781641240.145495, type 1 (EV_KEY), code 276 (**BTN_EXTRA**), value 0"
- **BTN_EXTRA** will be the keycode to my PTT button
- Now to get the device path run `udevadm info --query=all --name=/dev/input/event18 | grep /dev/input/by-id/`
- **!! REPLACE !!** the /dev/input/event**18** with the number you have gotten from `evtest`
- You should see something like this - if you don't, create a GitHub issue

```

E: DEVLINKS=/dev/input/by-path/pci-0000:00:14.0-usb-0:9:1.2-event-mouse /dev/input/by-path/pci-0000:00:14.0-usbv2-0:9:1.2-event-mouse /dev/input/by-id/usb-Compx_2.4G_Wireless_8K_Receiver-if02-event-mouse

````

- The path that starts with `/dev/input/by-id/` should be highlighted red, that is the one we need
- For this example it's `/dev/input/by-id/usb-Compx_2.4G_Wireless_8K_Receiver-if02-event-mouse`
- Save both the device path and keycode to a text editor

### Step 3 - Installing pttd

- Check if you have `brew` installed by running

```bash
brew -v
```

If it returns something like `Homebrew 6.0.2`, good, you have brew installed

If something like `bash: brew: command not found`, install brew from https://brew.sh

- Tap the repository by running

```bash
brew tap atmelmicro/pttd
```

- Trust the pttd package

```bash
brew trust --formula atmelmicro/pttd/pttd
```

- Install pttd `brew install pttd`

```bash
brew install pttd
```

### Step 4 - Creating a config

- Run `mkdir ~/.config/pttd/`
- Then run `nano ~/.config/pttd/config.yaml`
- Example config:

```yaml
keycode: BTN_EXTRA # Change this to your desired keycode
device: "/dev/input/by-id/usb-Compx_2.4G_Wireless_8K_Receiver-if02-event-mouse" # Change this to your device path
delay: 200 # When you stop pressing your PTT key, pttd will wait x ms until it cuts you off
volume: -2 # Volume control for the ping of PTT
kde: false # Enable this to use KDE icons instead of GNOME ones
```

- Press `Ctrl+X`, then `Y` then `Enter`

### Step 5 - running pttd

- Run `pttd` in the terminal
- If all goes well, you should see something like this

```
Input device name: "Compx 2.4G Wireless 8K Receiver"
Input device ID: bus 0x3 vendor 0x260d product 0x1154
```

- When you go into your sound settings, you should see that your mic is muted, until you press your PTT key
- If you have AppIndicator support in Gnome, or run KDE, you should see a tray icon
  - Pressing it you can disable the program or quit it
- To run it in the background and on startup, run `brew services start pttd`

**That's it, you have pttd running**

If you have encountered any issues, create a GitHub issue

## Step 6 - Running without brew (for advanced users)

For those who do not want to install homebrew, there are prebuilt binaries in the GitHub releases

## The why

On Wayland, there is no way to have global push-to-talk. This has been a big issue (apart from anti-cheat) for me that has prevented me from gaming on Linux.

The global shortcuts API has been released, and it does somewhat solve this issue, but Discord has not implemented it. The API also has its issues like no mouse support.

This method is not ideal for people that want to have push-to-talk to Discord, but an open mic for their screen recording.

## Credits

- [wayland-push-to-talk-fix](https://github.com/Rush/wayland-push-to-talk-fix) by Rush - the whole input detection is taken from this project
- [adwaita-icon-theme](https://gitlab.gnome.org/GNOME/adwaita-icon-theme) by the GNOME Team - icons
- [breeze-icons](https://github.com/KDE/breeze-icons/tree/master) by the KDE team - icons for KDE
- [New Notification 040](https://pixabay.com/sound-effects/technology-new-notification-040-493469/) by Universfield - sounds
- [evdev](https://github.com/grafov/evdev) by grafov - Linux device library
- [beep](github.com/gopxl/beep) by gopxl - sound library
- [systray](github.com/gogpu/systray) by gogpu - system tray library

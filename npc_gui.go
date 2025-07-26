package main

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/kardianos/service"
)

const (
	prefServerAddr = "server_address"
	prefVkey       = "vkey"
	prefAutoStart  = "auto_start"
)

// CustomTheme defines a simple custom theme
type CustomTheme struct{}

func (m CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xff}
	case theme.ColorNameForeground:
		return color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case theme.ColorNamePrimary:
		return color.RGBA{R: 0x00, G: 0x7b, B: 0xff, A: 0xff} // Bootstrap primary blue
	case theme.ColorNameButton:
		return color.RGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	case theme.ColorNameInputBackground:
		return color.RGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	case theme.ColorNamePlaceHolder:
		return color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
	case theme.ColorNameScrollBar:
		return color.RGBA{R: 0x60, G: 0x60, B: 0x60, A: 0xff}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (m CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func (m CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

type program struct {
	executablePath string
	serverAddr     string
	vkey           string
	cmd            *exec.Cmd
	outputArea     *widget.Entry
	wg             *sync.WaitGroup
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	p.outputArea.SetText(fmt.Sprintf("Starting NPC with server: %s, vkey: %s\n", p.serverAddr, p.vkey))

	// Check if npc executable exists
	npcPath := "./npc"
	if _, err := os.Stat(npcPath); os.IsNotExist(err) {
		p.outputArea.SetText(fmt.Sprintf("Error: npc executable not found at %s. Please ensure it's in the same directory as npc_gui.\n", npcPath))
		return
	}

	p.cmd = exec.Command(npcPath, "-server", p.serverAddr, "-vkey", p.vkey)

	var err error
	stdoutPipe, err := p.cmd.StdoutPipe()
	if err != nil {
		p.outputArea.SetText(fmt.Sprintf("Error creating stdout pipe: %v\n", err))
		return
	}
	stderrPipe, err := p.cmd.StderrPipe()
	if err != nil {
		p.outputArea.SetText(fmt.Sprintf("Error creating stderr pipe: %v\n", err))
		return
	}

	if err := p.cmd.Start(); err != nil {
		p.outputArea.SetText(fmt.Sprintf("Error starting NPC: %v\n", err))
		return
	}

	p.wg.Add(2) // Two goroutines for stdout and stderr

	// Read stdout
	go func() {
		defer p.wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				p.outputArea.SetText(p.outputArea.Text + string(buf[:n]))
			}
			if err != nil {
				if err != io.EOF {
					p.outputArea.SetText(p.outputArea.Text + fmt.Sprintf("Error reading stdout: %v\n", err))
				}
				return
			}
		}
	}()

	// Read stderr
	go func() {
		defer p.wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				p.outputArea.SetText(p.outputArea.Text + string(buf[:n]))
			}
			if err != nil {
				if err != io.EOF {
					p.outputArea.SetText(p.outputArea.Text + fmt.Sprintf("Error reading stderr: %v\n", err))
				}
				return
			}
		}
	}()

	go func() {
		err := p.cmd.Wait()
		if err != nil {
			p.outputArea.SetText(p.outputArea.Text + fmt.Sprintf("NPC exited with error: %v\n", err))
		} else {
			p.outputArea.SetText(p.outputArea.Text + "NPC exited successfully.\n")
		}
		p.wg.Wait() // Ensure all output is processed before marking as finished
	}()
}

func (p *program) Stop(s service.Service) error {
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		p.wg.Wait() // Wait for goroutines to finish
	}
	return nil
}

func main() {
	a := app.NewWithID("io.ehang.nps.npcgui") // Use an application ID for preferences
	w := a.NewWindow("NPC Client GUI")

	// Set custom theme
	a.Settings().SetTheme(&CustomTheme{})

	serverEntry := widget.NewEntry()
	serverEntry.SetPlaceHolder("Enter server address (e.g., 192.168.1.1:8024)")

	vkeyEntry := widget.NewEntry()
	vkeyEntry.SetPlaceHolder("Enter vkey")

	outputArea := widget.NewMultiLineEntry()
	outputArea.SetPlaceHolder("NPC client output will appear here...")
	outputArea.Wrapping = fyne.TextWrapBreak
	outputArea.Disable() // Make it read-only

	// Load saved preferences
	serverEntry.SetText(a.Preferences().String(prefServerAddr))
	vkeyEntry.SetText(a.Preferences().String(prefVkey))

	// Auto-start checkbox
	autoStartCheck := widget.NewCheck("Start with system", func(checked bool) {
		a.Preferences().SetBool(prefAutoStart, checked)
		// options := make(service.KeyValue) // Empty options
		// executablePath, err := os.Executable()
		// if err != nil {
		// 	dialog.ShowError(fmt.Errorf("Could not get executable path: %v", err), w)
		// 	return
		// }
		// svcConfig := &service.Config{
		// 	Name:        "NPCGUI",
		// 	DisplayName: "NPC GUI Client",
		// 	Description: "NPS Client GUI for easy management.",
		// 	Arguments:   []string{"--autostart"}, // Pass a flag to indicate autostart
		// 	Executable:  executablePath,
		// 	Option:      options,
		// }

		// prg := &program{}
		// s, err := service.New(prg, svcConfig)
		// if err != nil {
		// 	dialog.ShowError(fmt.Errorf("Error creating service: %v", err), w)
		// 	return
		// }

		// if checked {
		// 	err = s.Install()
		// 	if err != nil {
		// 		dialog.ShowError(fmt.Errorf("Error installing service: %v", err), w)
		// 		return
		// 	}
		// 	dialog.ShowInformation("Auto-start", "NPC GUI Client will now start with your system.", w)
		// } else {
		// 	err = s.Uninstall()
		// 	if err != nil {
		// 		dialog.ShowError(fmt.Errorf("Error uninstalling service: %v", err), w)
		// 		return
		// 	}
		// 	dialog.ShowInformation("Auto-start", "NPC GUI Client will no longer start with your system.", w)
		// }
	})
	autoStartCheck.SetChecked(a.Preferences().Bool(prefAutoStart))

	var currentProgram program
	currentProgram.outputArea = outputArea
	currentProgram.wg = &sync.WaitGroup{}

	startBtn := widget.NewButton("Start NPC", func() {
		serverAddr := serverEntry.Text
		vkey := vkeyEntry.Text

		if serverAddr == "" || vkey == "" {
			dialog.ShowError(fmt.Errorf("Server address and vkey cannot be empty."), w)
			return
		}

		// Save preferences
		a.Preferences().SetString(prefServerAddr, serverAddr)
		a.Preferences().SetString(prefVkey, vkey)

		currentProgram.serverAddr = serverAddr
		currentProgram.vkey = vkey

		// Kill existing process if running
		if currentProgram.cmd != nil && currentProgram.cmd.Process != nil {
			_ = currentProgram.cmd.Process.Kill()
			currentProgram.wg.Wait() // Wait for goroutines to finish
			outputArea.SetText("Previous NPC process stopped.\n")
		}

		currentProgram.run()
	})

	stopBtn := widget.NewButton("Stop NPC", func() {
		if currentProgram.cmd != nil && currentProgram.cmd.Process != nil {
			err := currentProgram.cmd.Process.Kill()
			if err != nil {
				dialog.ShowError(fmt.Errorf("Error stopping NPC: %v", err), w)
			} else {
				outputArea.SetText(outputArea.Text + "NPC process stopped.\n")
			}
			currentProgram.wg.Wait() // Wait for goroutines to finish
		} else {
			outputArea.SetText("NPC is not running.\n")
		}
	})

	// Create a title label
	titleLabel := canvas.NewText("NPC Client Configuration", color.White)
	titleLabel.TextSize = 20
	titleLabel.Alignment = fyne.TextAlignCenter

	// Layout for input fields and buttons
	inputForm := container.New(layout.NewFormLayout(),
		widget.NewLabel("Server Address:"), serverEntry,
		widget.NewLabel("Vkey:"), vkeyEntry,
	)

	buttonRow := container.NewHBox(
		layout.NewSpacer(), // Push buttons to center/right
		startBtn,
		stopBtn,
		layout.NewSpacer(),
	)

	// Main content layout
	content := container.NewVBox(
		titleLabel,
		layout.NewSpacer(),
		inputForm,
		layout.NewSpacer(),
		autoStartCheck,
		layout.NewSpacer(),
		buttonRow,
		layout.NewSpacer(),
		widget.NewLabel("Output:"),
		container.NewScroll(outputArea),
	)

	w.SetContent(container.NewBorder(
		nil, // Top
		nil, // Bottom
		nil, // Left
		nil, // Right
		container.NewPadded(content),
	))

	w.Resize(fyne.NewSize(600, 500)) // Slightly larger window
	w.ShowAndRun()
}
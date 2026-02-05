package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/atotto/clipboard"
)

func Copy(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		return tryFallback(text)
	}
	return nil
}

func tryFallback(text string) error {
	switch runtime.GOOS {
	case "linux":
		return copyLinux(text)
	case "darwin":
		return copyMacOS(text)
	case "windows":
		return copyWindows(text)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func copyLinux(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	
	if err := cmd.Start(); err != nil {
		cmd = exec.Command("wl-copy")
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("no clipboard utility found (install xclip or wl-clipboard)")
		}
	}
	
	stdin.Write([]byte(text))
	stdin.Close()
	return cmd.Wait()
}

func copyMacOS(text string) error {
	cmd := exec.Command("pbcopy")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	
	if err := cmd.Start(); err != nil {
		return err
	}
	
	stdin.Write([]byte(text))
	stdin.Close()
	return cmd.Wait()
}

func copyWindows(text string) error {
	cmd := exec.Command("clip")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	
	if err := cmd.Start(); err != nil {
		return err
	}
	
	stdin.Write([]byte(text))
	stdin.Close()
	return cmd.Wait()
}
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"unsafe"

	"github.com/ham-agents/ham-agents/go/internal/ipc"
)

// runWithPTY runs a provider command inside a pseudo-terminal so the user gets
// a fully interactive session while ham simultaneously reads the output and
// forwards lines to the daemon for state inference.
func runWithPTY(
	ctx context.Context,
	client *ipc.Client,
	agentID string,
	providerBin string,
	providerName string,
	projectPath string,
) error {
	// Open a new PTY pair.
	ptmx, ttyPath, err := openPTY()
	if err != nil {
		return fmt.Errorf("open pty: %w", err)
	}
	defer ptmx.Close()

	tty, err := os.OpenFile(ttyPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open tty %s: %w", ttyPath, err)
	}
	defer tty.Close()

	// Match the PTY size to the real terminal.
	if err := inheritTerminalSize(ptmx); err != nil {
		// Non-fatal — the session will still work, just maybe wrong size.
		fmt.Fprintf(os.Stderr, "ham: warning: could not set terminal size: %v\n", err)
	}

	// Propagate SIGWINCH (terminal resize) to the PTY.
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		for range sigwinch {
			_ = inheritTerminalSize(ptmx)
		}
	}()
	defer func() {
		signal.Stop(sigwinch)
		close(sigwinch)
	}()

	// Put the real terminal into raw mode so keystrokes pass through directly.
	// Skip if stdin is not a terminal (e.g. running inside a pipe or socket).
	var oldState *termios
	if isTerminal(os.Stdin.Fd()) {
		oldState, err = makeRaw(os.Stdin.Fd())
		if err != nil {
			return fmt.Errorf("set raw mode: %w", err)
		}
	}
	defer restoreTerminal(os.Stdin.Fd(), oldState)

	// Build the provider command using the PTY as stdin/stdout/stderr.
	cmd := exec.Command(providerBin)
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	cmd.Env = append(os.Environ(),
		"HAM_AGENT_ID="+agentID,
	)
	if projectPath != "" {
		cmd.Dir = projectPath
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start provider: %w", err)
	}
	// Close the slave side in the parent — the child owns it now.
	tty.Close()

	// Copy stdin → ptmx (user input to provider).
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	// Copy ptmx → stdout, and tee output to daemon.
	go func() {
		buf := make([]byte, 4096)
		var lineBuf strings.Builder
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				// Show to user.
				_, _ = os.Stdout.Write(buf[:n])

				// Accumulate lines and send to daemon.
				for _, b := range buf[:n] {
					if b == '\n' || b == '\r' {
						line := strings.TrimSpace(lineBuf.String())
						if line != "" {
							_ = client.RecordOutput(context.Background(), agentID, line)
						}
						lineBuf.Reset()
					} else {
						lineBuf.WriteByte(b)
					}
				}
			}
			if err != nil {
				break
			}
		}
	}()

	return cmd.Wait()
}

// openPTY opens a new pseudo-terminal pair using POSIX calls.
func openPTY() (ptmx *os.File, ttyPath string, err error) {
	ptmxFD, err := syscall.Open("/dev/ptmx", syscall.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, "", fmt.Errorf("open /dev/ptmx: %w", err)
	}
	ptmx = os.NewFile(uintptr(ptmxFD), "/dev/ptmx")

	// grantpt
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(ptmxFD), syscall.TIOCPTYGRANT, 0); errno != 0 {
		ptmx.Close()
		return nil, "", fmt.Errorf("grantpt: %v", errno)
	}

	// unlockpt
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(ptmxFD), syscall.TIOCPTYUNLK, 0); errno != 0 {
		ptmx.Close()
		return nil, "", fmt.Errorf("unlockpt: %v", errno)
	}

	// ptsname — get the path of the slave device
	nameBuf := make([]byte, 128)
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(ptmxFD), syscall.TIOCPTYGNAME, uintptr(unsafe.Pointer(&nameBuf[0]))); errno != 0 {
		ptmx.Close()
		return nil, "", fmt.Errorf("ptsname: %v", errno)
	}
	ttyPath = strings.TrimRight(string(nameBuf), "\x00")

	return ptmx, ttyPath, nil
}

type termios syscall.Termios

func makeRaw(fd uintptr) (*termios, error) {
	var old termios
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TIOCGETA, uintptr(unsafe.Pointer(&old)), 0, 0, 0); errno != 0 {
		return nil, errno
	}

	raw := old
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TIOCSETA, uintptr(unsafe.Pointer(&raw)), 0, 0, 0); errno != 0 {
		return nil, errno
	}

	return &old, nil
}

func isTerminal(fd uintptr) bool {
	var t termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TIOCGETA, uintptr(unsafe.Pointer(&t)), 0, 0, 0)
	return errno == 0
}

func restoreTerminal(fd uintptr, state *termios) {
	if state == nil {
		return
	}
	_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TIOCSETA, uintptr(unsafe.Pointer(state)), 0, 0, 0)
}

func inheritTerminalSize(ptmx *os.File) error {
	var ws struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws))); errno != 0 {
		return errno
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, ptmx.Fd(), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws))); errno != 0 {
		return errno
	}
	return nil
}

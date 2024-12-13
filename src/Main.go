package main

import (
    "fmt"
    "os"
    "syscall"
    "os/signal"
    "golang.org/x/term" 
    "strings"
    "sync"
)

func ClearScreen() {
    fmt.Fprintf(os.Stdout, "\033[2J")
}

func SaveCursor() {
    fmt.Fprintf(os.Stdout, "\033[s")
}

func RestoreCursor() {
    fmt.Fprintf(os.Stdout, "\033[u")
}

func EnterAltScreen() {
    fmt.Fprintf(os.Stdout, "\033[?1049h")
}

func ExitAltScreen() {
    fmt.Fprintf(os.Stdout, "\033[?1049l")
}

func MoveCursorDown() {
    fmt.Fprintf(os.Stdout, "\033[1B")
}

func MoveCursorUp() {
    fmt.Fprintf(os.Stdout, "\033[1A")
}

func MoveCursorToStartLine() {
    fmt.Fprintf(os.Stdout, "\033[1G")
}

func MoveHome() {
    fmt.Fprintf(os.Stdout, "\033[H")
}

func enterTUI() {
    SaveCursor()
    EnterAltScreen()
    ClearScreen()
}

func exitTUI() {
    ExitAltScreen()
    RestoreCursor()
}

func PrintLine(line string) {
    // Strip trailing newlines
    line = strings.TrimRight(line, "\n")
    fmt.Fprintf(os.Stdout, line)
    MoveCursorDown()
    // Move cursor to beginning of the line
    MoveCursorToStartLine()
}

type TermState struct {
    mu sync.Mutex
    Rows int
    Cols int
}

func (t *TermState) GetSize() (int, int) {
    t.mu.Lock()
    defer t.mu.Unlock()
    return t.Rows, t.Cols
}

func main() {
// Ensure terminal state is restored on exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

    sigWinCh := make(chan os.Signal, 1)
    signal.Notify(sigWinCh, syscall.SIGWINCH)

    oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }

    defer func() {
        term.Restore(int(os.Stdin.Fd()), oldState)
        exitTUI()
    }()

	go func() {
		<-c
		exitTUI()
		os.Exit(0)
	}()


	// Enter TUI mode
	enterTUI()

    // Get terminal size
    cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }

    termState := TermState{Rows: rows, Cols: cols}
    go func(termState *TermState) {
        for range sigWinCh {
            cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
            if err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
            }

            termState.mu.Lock()
            termState.Rows = rows
            termState.Cols = cols
            termState.mu.Unlock()

            PrintLine(fmt.Sprintf("Resized to rows: %v, cols: %v\n", rows, cols))
        }
    }(&termState)

    MoveHome()
    PrintLine(fmt.Sprintf("rows: %v, cols: %v\n", rows, cols))
    // termState := TermState{Rows: rows, Cols: cols}

	// Simulate a TUI by printing to the alternate screen
	PrintLine("Welcome to the TUI!")
	PrintLine("Press Ctrl+C or q to exit.")

    buf := make([]byte, 1)
    for {
        n, err := os.Stdin.Read(buf)

        if err != nil || n == 0 {
            fmt.Fprintf(os.Stderr, "error: %v\n", err)
            break
        }

        switch buf[0] {
        case 3: // Ctrl+C
            return
        case 'q':
            return
        case 'j':
            PrintLine("down\n")
        case 'k':
            PrintLine("up\n")
        default:
            message := fmt.Sprintf("unknown key: %v\n", buf[0])
            PrintLine(message)
        }
    }
}

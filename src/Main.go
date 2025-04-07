package main

import (
    "fmt"
    "os"
    "syscall"
    "os/signal"
    "golang.org/x/term"
    "strings"
    "sync"
	"crypto/sha256"
	"io"
	"os/exec"
	"net/mail"
	"mime/multipart"
	"mime"
	"encoding/base64"
	"bytes"
	"golang.org/x/text/encoding/charmap"
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

type GenericHeader interface {
	Get(key string) string
}

func main() {

	// Parse CLI args
	subcommand := ""
	i := 1
	var file string

	for i < len(os.Args) {
		arg := os.Args[i]
		i++

		if arg == "-h" || arg == "--help" {
			fmt.Println("Usage: memail [upload <file> | parse <file>]")
			os.Exit(0)
		} else if arg == "upload" {
			// Check that there is a single file argument to upload
			if i >= len(os.Args) {
				fmt.Println("Error: upload subcommand requires a file argument")
				os.Exit(1)
			} else if i < len(os.Args) - 1 {
				fmt.Println("Error: upload subcommand only accepts a single file argument")
				os.Exit(1)
			}

			subcommand = "upload"
			file = os.Args[i]
		} else if arg == "parse" {
			// Check that there is a single file argument to parse
			if i >= len(os.Args) {
				fmt.Println("Error: parse subcommand requires a file argument")
				os.Exit(1)
			} else if i < len(os.Args) - 1 {
				fmt.Println("Error: parse subcommand only accepts a single file argument")
				os.Exit(1)
			}

			subcommand = "parse"
			file = os.Args[i]
		}
	}


	if subcommand == "upload" {
		// Upload the file using b2-linux CLI
		// b2-linux upload-file <bucketName> <localFilePath> <b2FileName>

		// Get bucket name from MEMAIL_BUCKET env var
		bucketName, ok := os.LookupEnv("MEMAIL_BUCKET")
		if !ok {
			fmt.Println("Error: MEMAIL_BUCKET environment variable not set")
			os.Exit(1)
		}

		// Get the SHA-256 hash of the file
		file, err := os.Open(file)
		if err != nil {
			fmt.Println("Error: could not open file")
			os.Exit(1)
		}

		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			fmt.Println("Error: could not hash file")
			os.Exit(1)
		}

		checksum := hash.Sum(nil)

		// Reset the file pointer to the beginning
		file.Seek(0, 0)

		// Parse the file as an email as a test
		_, err = mail.ReadMessage(file)
		if err != nil {
			fmt.Println("Error: could not parse file as email")
			os.Exit(1)
		}

		// Upload the file with the checksum as the name, .eml extension
		b2Cmd := exec.Command("b2", "upload-file", bucketName, file.Name(), fmt.Sprintf("%x.eml", checksum))

		app_key_id, ok := os.LookupEnv("MEMAIL_APP_KEY_ID")
		if !ok {
			fmt.Println("Error: MEMAIL_APP_KEY_ID environment variable not set")
			os.Exit(1)
		}

		app_key, ok := os.LookupEnv("MEMAIL_APP_KEY")
		if !ok {
			fmt.Println("Error: MEMAIL_APP_KEY environment variable not set")
			os.Exit(1)
		}

		// Set the B2_APPLICATION_KEY_ID and B2_APPLICATION_KEY environment variables, using the MEMAIL_APP_KEY_ID and MEMAIL_APP_KEY values
		b2Cmd.Env = append(b2Cmd.Env, fmt.Sprintf("B2_APPLICATION_KEY_ID=%s", app_key_id))
		b2Cmd.Env = append(b2Cmd.Env, fmt.Sprintf("B2_APPLICATION_KEY=%s", app_key))

		b2Cmd.Stdout = os.Stdout
		b2Cmd.Stderr = os.Stderr

		err = b2Cmd.Run()
		if err != nil {
			fmt.Println("Error: could not upload file")
			os.Exit(1)
		}

		os.Exit(0)
	} else if subcommand == "parse" {
		// Parse the file as an email
		file, err := os.Open(file)
		if err != nil {
			fmt.Println("Error: could not open file")
			os.Exit(1)
		}

		message, err := mail.ReadMessage(file)
		if err != nil {
			fmt.Println("Error: could not parse file as email")
			os.Exit(1)
		}

		from := message.Header.Get("From")
		to := message.Header.Get("To")
		subject := message.Header.Get("Subject")
		dateTime := message.Header.Get("Date")
		contentType := message.Header.Get("Content-Type")

		fmt.Printf("From: %s\n", from)
		fmt.Printf("To: %s\n", to)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Date: %s\n", dateTime)
		fmt.Printf("Content-Type: %s\n", contentType)
		fmt.Printf("\n")

		HandleContent(message.Body, message.Header)
		os.Exit(0)
	}


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

func HandleContent(reader io.Reader, header GenericHeader) {
	contentType := header.Get("Content-Type")
	mimetype, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		fmt.Println("Error: could not parse media type")
		os.Exit(1)
	}

	// fmt.Fprintf(os.Stderr, "Handling mime type: %s\n", mimetype)
	if mimetype == "multipart/alternative" {
		// HandleMultiPartAlterative(reader, params["boundary"])
		HandleMultiPart(reader, params["boundary"])
	} else if mimetype == "multipart/related" {
		HandleMultiPart(reader, params["boundary"])
	} else if mimetype == "text/plain" {
		encoding := header.Get("Content-Transfer-Encoding")
		HandleTextPlain(reader, encoding, header)
	} else if mimetype == "multipart/mixed" {
		boundary := params["boundary"]
		HandleMultiPart(reader, boundary)
	} else {
		fmt.Fprintf(os.Stdout, "Error: unsupported content type: %s\n", contentType)
	}
}

func HandleTextPlain(reader io.Reader, contentTransferEncoding string, genericHeader GenericHeader) {
	body, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println("Error: could not read body")
		os.Exit(1)
	}

	// Decode the body if it is base64 encoded
	if contentTransferEncoding == "base64" {
		reader := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(body))
		decoded, err := io.ReadAll(reader)
		if err != nil {
			fmt.Println("Error: could not decode body")
			os.Exit(1)
		}

		fmt.Printf("%s\n", decoded)
		// Future: Check charset
		// charset, ok := altParams["charset"]
		// if ok {
		// if strings.ToLower(charset) == "utf-8" {

		// }
		// } else {
		// fmt.Printf("%s\n", decoded)
		// }
	} else {

		// Check the charset
		contentType := genericHeader.Get("Content-Type")
		_, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			fmt.Println("Error: could not parse content type")
			os.Exit(1)
		}

		charset, ok := params["charset"]
		if ok {
			if strings.ToLower(charset) == "utf-8" || strings.ToLower(charset) == "us-ascii" {
				// Print the body as is
				fmt.Printf("%s\n", body)
			} else if strings.ToLower(charset) == "windows-1256" {
				// Convert the body to UTF-8
				decoder := charmap.Windows1256.NewDecoder()
				decoded, err := decoder.String(string(body))
				if err != nil {
					fmt.Println("Error: could not decode body")
					os.Exit(1)
				}

				fmt.Printf("%s\n", decoded)
			} else if strings.ToLower(charset) == "windows-1252" {
				decoder := charmap.Windows1252.NewDecoder()
				decoded, err := decoder.String(string(body))
				if err != nil {
					fmt.Println("Error: could not decode body")
					os.Exit(1)
				}
				fmt.Printf("%s\n", decoded)
			} else {
				fmt.Printf("Error: unsupported charset: %s\n", charset)
			}
		} else {
			// Print the body as is
			fmt.Printf("%s\n", body)
		}
	}
}

func HandleMultiPart(body io.Reader, boundary string) {
	mr := multipart.NewReader(body, boundary)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error: could not read part")
			os.Exit(1)
		}

		contentDisposition := p.Header.Get("Content-Disposition")
		if contentDisposition != "" {
			// Check for attachment
			dispositionType, params, err := mime.ParseMediaType(contentDisposition)
			if err != nil {
				fmt.Println("Error: could not parse content disposition")
				os.Exit(1)
			}
			if dispositionType == "attachment" {
				filename, ok := params["filename"]
				if !ok {
					fmt.Println("Error: could not find filename in content disposition")
					os.Exit(1)
				}

				contentType := p.Header.Get("Content-Type")
				// Check for content type


				fmt.Printf("Attachment: %s; %s\n", filename, contentType)
				// Save the attachment to a file
				// outFile, err := os.Create(filename)
				// if err != nil {
					// fmt.Println("Error: could not create file")
					// os.Exit(1)
				// }
				// defer outFile.Close()

				// _, err = io.Copy(outFile, p)
				// if err != nil {
					// fmt.Println("Error: could not save attachment")
					// os.Exit(1)
				// }
			}
		} else {
			HandleContent(p, p.Header)
		}
	}
}

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rivo/tview"
    "github.com/gdamore/tcell/v2"
    "net/mail"
    "bytes"
)

func main2() {
	// Get files from command-line arguments
	files := os.Args[1:]
	if len(files) == 0 {
		fmt.Println("Usage: go run main.go <file1> <file2> ...")
		os.Exit(1)
	}

	// Create a new application
	app := tview.NewApplication()

	// File table
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)


    // Files are expected to be email files
    // Parse the email files, and display the Date, From, and Subject fields in the table
    row := 0
    for _, file := range files {
        // Read the file content
        content, err := os.ReadFile(file)
        if err != nil {
            // Skip if there is an error reading the file
            continue
        }

        // Parse the email
        msg, err := mail.ReadMessage(bytes.NewReader(content))
        if err != nil {
            // Skip if there is an error parsing the email
            continue
        }

        // Get the Date, From, and Subject fields
        date := msg.Header.Get("Date")
        from := msg.Header.Get("From")
        subject := msg.Header.Get("Subject")

        // Add the fields to the table, first column has max width of 30
        table.SetCell(row, 0, tview.NewTableCell(date).SetMaxWidth(30))
        table.SetCellSimple(row, 1, from)
        table.SetCellSimple(row, 2, subject)

        row++
    }

	// // Populate the table with filenames
	// for i, file := range files {
		// table.SetCell(i, 0, tview.NewTableCell(file).
			// SetSelectable(true).
			// SetAlign(tview.AlignLeft))
	// }

	// Open file in $EDITOR or Neovim
	openInEditor := func(filePath string) {
		// Read the file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading file: %s\n", err.Error())
			return
		}

		// Create a temporary file
		tempFile, err := os.CreateTemp("", "email-*.txt")
		if err != nil {
			fmt.Printf("Error creating temporary file: %s\n", err.Error())
			return
		}
		defer os.Remove(tempFile.Name()) // Clean up temp file after editor closes

		// Write the content to the temporary file
		if _, err := tempFile.Write(content); err != nil {
			fmt.Printf("Error writing to temporary file: %s\n", err.Error())
			return
		}
		tempFile.Close()

		// Determine the editor to use
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nvim" // Default to Neovim
		}

		// Launch the editor
		cmd := exec.Command(editor, tempFile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("Error launching editor: %s\n", err.Error())
			return
		}
	}

	// Navigation and selection
	table.SetSelectedFunc(func(row, column int) {
		// Get selected file
		selectedFile := files[row]

		// Open the file in the editor
		openInEditor(selectedFile)
	})

	// Keybindings for navigation
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j':
            currRow, _ := table.GetSelection()
			table.Select(currRow + 1, 0)
			return nil
		case 'k':
            currRow, _ := table.GetSelection()
			table.Select(currRow-1, 0)
			return nil
		
        case 'q':
            app.Stop()
        }
		return event
	})

	// Set the initial view to the table
	if err := app.SetRoot(table, true).Run(); err != nil {
		panic(err)
	}
}

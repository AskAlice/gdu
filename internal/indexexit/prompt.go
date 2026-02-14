package indexexit

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ServeCommand returns the command to run the search server for an index (one binary: gdu)
func ServeCommand(indexPath string, port int) string {
	if port == 0 {
		port = 8765
	}
	return fmt.Sprintf("gdu serve -i %q -p %d", indexPath, port)
}

// OnExitPrompt asks whether to start search. Returns true to start serve, false otherwise.
// If false, call PrintServeCommand after saving the index.
func OnExitPrompt(indexPath string, port int) bool {
	fmt.Fprint(os.Stderr, "\nSave index and open search? [y/N] ")
	rd := bufio.NewReader(os.Stdin)
	line, _ := rd.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

// PrintServeCommand prints the command to run the search server later
func PrintServeCommand(indexPath string, port int) {
	cmd := ServeCommand(indexPath, port)
	fmt.Fprintf(os.Stderr, "\nIndex saved. To search later, run:\n  %s\n", cmd)
}

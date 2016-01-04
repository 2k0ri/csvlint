package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"os"
	"encoding/csv"
	"bufio"
	"regexp"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

func printCsv(w io.Writer, row []string) (e error) {
	r := strings.NewReplacer(
		`\"`, `""`, // \" is not genuine escape in csv format, so convert manually
		`"`, `""`,
	)

	sep := ""

	for _, cell := range row {
		_, err := io.WriteString(w, sep + `"` + r.Replace(cell) + `"`)
		if err != nil {
			fmt.Errorf(err.Error())
			continue
		}
		sep = ","
	}
	_, err := io.WriteString(w, "\n")
	if err != nil {
		fmt.Errorf(err.Error())
	}
	return
}

func printTsv(w io.Writer, row []string) (e error) {
	r := strings.NewReplacer(
		"\t", "\\t",
	)

	sep := ""

	for _, cell := range row {
		_, err := io.WriteString(w, sep + r.Replace(cell))
		if err != nil {
			fmt.Errorf(err.Error())
			continue
		}
		sep = "\t"
	}
	_, err := io.WriteString(w, "\n")
	if err != nil {
		fmt.Errorf(err.Error())
	}
	return
}


// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		removeTab     bool
		removeNewline bool
		removeSpace   bool
		tsv           bool
		file          string

		version bool
	)

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)

	flags.BoolVar(&removeTab, "remove-tab", false, "remove tab")
	flags.BoolVar(&removeTab, "t", false, "remove tab(Short)")
	flags.BoolVar(&removeNewline, "remove-newline", false, "remove newline in column")
	flags.BoolVar(&removeNewline, "n", false, "remove newline in column(Short)")
	flags.BoolVar(&removeSpace, "remove-space", false, "remove sparse spaces")
	flags.BoolVar(&removeSpace, "s", false, "remove sparse spaces(Short)")
	flags.BoolVar(&tsv, "tsv", false, "output tsv")
	flags.BoolVar(&tsv, "T", false, "output tsv(Short)")
	flags.StringVar(&file, "file", "", "file")
	flags.StringVar(&file, "f", "", "file(Short)")

	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(cli.errStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	var fp *os.File
	if file == "" {
		fp = os.Stdin
	} else {
		var err error
		fp, err = os.Open(file)
		if err != nil {
			panic(err)
		}
		defer fp.Close()
	}

	replacerArgs := []string{
		"\u00A0", "\x20", // another type space
	}

	if removeTab {
		replacerArgs = append(replacerArgs, "\t", "")
	}

	if removeNewline {
		replacerArgs = append(replacerArgs, "\n", "", "\r", "")
	} else {
		replacerArgs = append(replacerArgs, "\n", "\\n", "\r", "\\r")
	}

	reTrS := regexp.MustCompile(`\s{2,}`)

	var printFunc func(io.Writer, []string) error
	if tsv {
		printFunc = printTsv
	} else {
		printFunc = printCsv
	}

	replacer := strings.NewReplacer(replacerArgs...)

	reader := csv.NewReader(fp)
	reader.LazyQuotes = true

	writer := bufio.NewWriter(os.Stdout)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Errorf(err.Error())
		}

		for i, v := range record {
			record[i] = replacer.Replace(v)
			if removeSpace {
				record[i] = strings.TrimSpace(reTrS.ReplaceAllString(record[i], " "))
			}
		}

		if err := printFunc(writer, record); err != nil {
			fmt.Errorf(err.Error())
		}
	}
	writer.Flush()

	return ExitCodeOK
}

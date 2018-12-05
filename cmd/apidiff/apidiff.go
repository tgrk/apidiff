package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/olekukonko/tablewriter"
	"github.com/tgrk/apidiff"
)

// Version of application.
const Version = "0.0.1"

var (
	// generic
	version = flag.Bool("v", false, "prints current program version")
	verbose = flag.Bool("verbose", false, "output basic progress")

	// commands
	recordCmd  = flag.Bool("record", false, "record a new API session")
	compareCmd = flag.Bool("compare", false, "compare recorded sessions against a URL")
	listCmd    = flag.Bool("list", false, "list all recorded API sessions")
	deleteCmd  = flag.Bool("del", false, "list all recorded API sessions")
	showCmd    = flag.Bool("show", false, "list all recorded API sessions")

	// command specific
	name      = flag.String("name", "", "name of session to be recorded")
	source    = flag.String("source", "", "source recorded session for comparison")
	target    = flag.String("target", "", "target recorded session for comparision")
	directory = flag.String("dir", "", "path where API calls are stored (default $HOME/.apidiff/)")
	headers   = flag.String("headers", "", "HTTP headers to use for API request (eg. Content-Type or Authorize)")
	excludes  = flag.String("excludes", "", "exclude specified HTTP headers from comparison (eg. Date or Authorize)")
)

func main() {
	flag.Parse()

	if flag.NFlag() == 0 {
		printErrorf("Usage: %s [OPTIONS] argument ...\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *version {
		fmt.Printf("Version: %s\n", Version)
		os.Exit(0)
	}

	// get storage directory (defaults to $HOME/.apidiff)
	var err error
	var directoryPath = *directory
	if directoryPath == "" {
		directoryPath, err = ensureDefaultDirectoryExists()
		if err != nil {
			printErrorf("Unable to create default directory due to %s!", err)
		}
	}

	options := apidiff.Options{
		Verbose: *verbose,
		Name:    *name,
	}

	ad := apidiff.New(directoryPath, options)

	if *listCmd {
		sessions, err := ad.List()
		if err != nil {
			printErrorf("Unable to list recorded sessions due to %s", err)
		}
		if len(sessions) == 0 {
			fmt.Println("No recorded sessions found")
		} else {
			rows := [][]string{}
			for _, session := range sessions {
				rows = append(rows, []string{
					session.Name,
					session.Path,
					session.Created.Format("2006-01-02 15:04:05"),
				})
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
			table.SetCenterSeparator("|")
			table.SetHeader([]string{"Name", "Path", "Created"})
			table.SetAutoMergeCells(true)
			table.AppendBulk(rows)
			table.Render()
		}
	}

	if *recordCmd || *compareCmd {
		// reads manifest from STDIN or path as last CLI arg
		reader := os.Stdin
		filename := ""
		if flag.NArg() > 0 {
			filename = flag.Arg(0)
			f, err := os.Open(filename)
			if err != nil {
				printErrorf("Unable to read source file %q!", filename, err)
			}
			defer f.Close()
			reader = f
		}

		if *recordCmd {
			if *name == "" {
				printErrorln("Missing session name (-name \"foo\")")
				os.Exit(1)
			}

			s := apidiff.RecordedSession{
				Name: *name,
			}
			m := apidiff.NewManifest()
			m.Parse(reader)

			fmt.Printf("DEBUG: session=%+v\n", s)
			fmt.Printf("DEBUG: manifest=%+v\n", *m)
		}
	}
}

func ensureDefaultDirectoryExists() (string, error) {
	dirPath, err := getDefaultDirectory()
	if err != nil {
		printErrorf("Unable to get default directory due to %s!", err)
		return "", err
	}
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return dirPath, os.MkdirAll(dirPath, os.ModePerm)
	}
	return dirPath, nil
}

func getDefaultDirectory() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return path.Join(usr.HomeDir, ".apidiff"), nil
}

func printErrorln(message string) {
	fmt.Fprintln(os.Stderr, message)
}

func printErrorf(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
}

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path"
	"time"

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
	directory = flag.String("dir", "", "path where API calls are stored (default $HOME/.apidiff/)")
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

	ui := apidiff.NewUI()

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
			ui.ListSessions(sessions, true)
		}
	}

	if *showCmd {
		sessionName := *name
		if flag.NArg() > 0 {
			sessionName = flag.Arg(0)
		}

		if sessionName == "" {
			printErrorln("Missing session name (-name \"foo\")")
			os.Exit(1)
		}

		session, err := ad.Show(sessionName)
		if err != nil {
			printErrorf("Unable to show recorded session due to %s", err)
			os.Exit(1)
		}

		ui.ShowSessionDetail(session)
	}

	if *deleteCmd {
		sessionName := *name
		if flag.NArg() > 0 {
			sessionName = flag.Arg(0)
		}
		if sessionName == "" {
			printErrorln("Missing session name (-name \"foo\")")
			os.Exit(1)
		}

		if err := ad.Delete(sessionName); err != nil {
			printErrorf("Unable to delete recorded session due to %s", err)
			os.Exit(1)
		}
	}

	if *recordCmd || *compareCmd {
		// reads manifest from STDIN or path as last CLI arg
		reader := bufio.NewReader(os.Stdin)
		filename := ""
		if flag.NArg() > 0 {
			filename = flag.Arg(0)
			f, err := os.Open(filename)
			if err != nil {
				printErrorf("Unable to read source file %q", filename)
				os.Exit(1)
			}
			defer f.Close()
			reader = bufio.NewReader(f)
		}

		if filename == "" || reader.Size() == 0 {
			printErrorln("No manifest supplied.")
			os.Exit(1)
		}

		if *recordCmd {
			if *name == "" {
				printErrorln("Missing session name (-name \"foo\")")
				os.Exit(1)
			}

			session := apidiff.RecordedSession{
				Name: *name,
			}

			manifest := apidiff.NewManifest()
			err := manifest.Parse(reader)
			if err != nil {
				printErrorf("Unable to parse source manifest due to %s", err)
				os.Exit(1)
			}

			start := time.Now()

			for _, request := range manifest.Requests {
				err = ad.Record(session.Name, request, nil)
				if err != nil {
					printErrorf("Unable to record session due to %s", err)
				}
			}

			if ad.Options.Verbose {
				elapsed := time.Since(start)

				fmt.Fprintf(os.Stdout, "Recording finished in %0.3f seconds...\n", elapsed.Seconds())
			}
		}

		if *compareCmd {
			if *name == "" {
				printErrorln("Missing source session name (-name \"foo\")")
				os.Exit(1)
			}

			sourceSession, err := ad.Show(*name)
			if err != nil {
				printErrorln("Missing session name (-name \"foo\")")
				os.Exit(1)
			}

			targetManifest := apidiff.NewManifest()
			err = targetManifest.Parse(reader)
			if err != nil {
				printErrorf("Unable to parse target manifest due to %s", err)
				os.Exit(1)
			}

			//TODO: compare also just using source and manifest input
			errors := ad.Compare(sourceSession, *targetManifest)
			if len(errors) > 0 {
				fmt.Printf("DEBUG: compare=%+v\n", errors)
			} else {
				printInfoln("Success. No differences found")
			}
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

func printInfoln(message string) {
	fmt.Fprintln(os.Stdout, message)
}

func printErrorln(message string) {
	fmt.Fprintln(os.Stderr, message)
}

func printErrorf(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
}

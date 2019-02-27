
[![CircleCI](https://circleci.com/gh/tgrk/apidiff/tree/master.svg?style=svg)](https://circleci.com/gh/tgrk/apidiff/tree/master)
[![codecov.io](https://codecov.io/github/tgrk/apidiff/coverage.svg?branch=master)](https://codecov.io/github/tgrk/apidiff?branch=master)


# API Diff

Records HTTP API (JSON based) calls and compares the them on both HTTP and JSON level. This is helpful when migrating or refactoring APIs to make sure your API contract did not change. It also stores basic performance metrics.

## Installation

As binary:
```bash
$ go get github.com/tgrk/apidiff
$ go install github.com/tgrk/apidiff/cmd/apidiff
```

As dependency:
```bash
$ go get gopkg.in/tgrk/apidiff.v1

```

## Usage

[![asciicast](https://asciinema.org/a/219377.svg)](https://asciinema.org/a/219377)

```bash
$ apidiff -h
Usage: apidiff [OPTIONS] argument ...

  -compare
    	compare recorded sessions against a URL
  -del
    	list all recorded API sessions
  -detail
    	view detail fo recorded API session
  -dir string
    	path where API calls are stored (default $HOME/.apidiff/)
  -list
    	list all recorded API sessions
  -name string
    	name of session to be recorded
  -record
    	record a new API session
  -show
    	list all recorded API sessions
  -v	prints current program version
  -verbose
    	output basic progress

```

### Record a new session

Reads [manifest file](examples/simple.yaml) from both CLI arguments and STDIN:

```bash
appidiff -record -name "foo" examples/simple.yaml
```

```bash
$ cat examples/simple.yaml | ./appidiff -record -name "foo"
```

### List all existing sessions
```bash
appidiff -list
```

### Show an existing session
```bash
appidiff -show "foo"
```

### Detail of first session interaction
```bash
appidiff -detail "foo" 1
```

### Delete an existing session
```bash
appidiff -del "foo"
```

### Compare against an existing sessions

Compare existing session against a manifest with other API:
```bash
appidiff -compare -name "bar" examples/simple.yaml
```

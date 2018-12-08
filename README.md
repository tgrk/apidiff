
[![CircleCI](https://circleci.com/gh/tgrk/apidiff/tree/master.svg?style=svg)](https://circleci.com/gh/tgrk/apidiff/tree/master)
[![codecov.io](https://codecov.io/github/tgrk/apidiff/coverage.svg?branch=master)](https://codecov.io/github/tgrk/apidiff?branch=master)


# API Diff

Records HTTP API (JSON based) calls and compares the them on both HTTP and JSON level. This is helpful when migrating or refactoring APIs to make sure your API contract did not change. It also stores basic performance metrics.

## Installation

```bash
$ go get github.com/tgrk/apidiff

```

## Usage

```bash
$ apidiff -h
Usage of apidiff:
  -compare
    	compare recorded sessions against a URL
  -del
    	list all recorded API sessions
  -dir string
    	path where API calls are stored (default $HOME/.apidiff/)
  -excludes string
    	exclude specified HTTP headers from comparison (eg. Date or Authorize)
  -headers string
    	HTTP headers to use for API request (eg. Content-Type or Authorize)
  -list
    	list all recorded API sessions
  -name string
    	name of session to be recorded
  -record
    	record a new API session
  -show
    	list all recorded API sessions
  -source string
    	source recorded session for comparison
  -target string
    	target recorded session for comparison
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

### Delete an existing session
```bash
appidiff -del "foo"
```

### Compare an existing sessions
```bash
appidiff -compare -source "foo" -target "bar"
```

Or compare existing session againgst a manifest with other API:
```bash
appidiff -compare -name "bar" examples/simple.yaml
```

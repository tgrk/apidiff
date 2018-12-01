
# API Diff
Records API calls and compares the

## Installation

## Usage

```bash
$ apidiff -h
Usage of ./apidiff:
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
    	target recorded session for comparision
  -v	prints current program version
  -verbose
    	output basic progress

```

### Record a new session

Reads from both CLI arguments and STDIN:
* single URL
* path to list of URLs in file (each on new line)
* STDIN

```bash
$ ./appidiff -record -name "foo" https://api.chucknorris.io/jokes/random
```

```bash
$ ./appidiff -record -name "foo" https://api.chucknorris.io/jokes/random
```

### List all existing sessions
```bash
$ ./appidiff -list
```

### Show an existing session
```bash
$ ./appidiff -show "foo"
```

### Delete an existing session
```bash
$ ./appidiff -del "foo"
```

### Compare an existing sessions agains an URI
```bash
$ ./appidiff -compare -name "foo" https://api.chucknorris.io/jokes/random
```


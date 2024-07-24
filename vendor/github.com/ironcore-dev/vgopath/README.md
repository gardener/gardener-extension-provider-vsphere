# vgopath

[![REUSE status](https://api.reuse.software/badge/github.com/ironcore-dev/vgopath)](https://api.reuse.software/info/github.com/ironcore-dev/vgopath)
[![GitHub License](https://img.shields.io/static/v1?label=License&message=Apache-2.0&color=blue)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/ironcore-dev/vgopath)](https://goreportcard.com/report/github.com/ironcore-dev/vgopath)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://makeapullrequest.com)

`vgopath` is a tool for module-enabled projects to set up a 'virtual' GOPATH for
legacy tools to run with (`kubernetes/code-generator` I'm looking at you...).

## Installation

The simplest way to install `vgopath` is by running

```shell
go install github.com/ironcore-dev/vgopath@latest
```

## Usage

`vgopath` has to be run from the module-enabled project root. It requires a
target directory to construct the virtual GOPATH.

Example usage could look like this:

```shell
# Create the target directory
mkdir -p my-vgopath

# Do the linking in my-vgopath
vgopath -o my-vgopath
```

Once done, the structure will look something like

```
my-vgopath
├── bin -> <GOPATH>/bin
├── pkg -> <GOPATH>/pkg
└── src -> various subdirectories
```

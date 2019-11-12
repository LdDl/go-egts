[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/LdDl/go-egts)
[![Build Status](https://travis-ci.com/LdDl/go-egts.svg?branch=master)](https://travis-ci.com/LdDl/go-egts)

# WORK IN PROGRESS: go-egts
EGTS (Era Glonass Telematics Standard) parsing via Golang

## PR's are welcome!

## TODO:
* Refactor code to make it more clear WIP
* Test coverage
* NEED TO BE REFACTORED ~~Example of TCP server (see /example/main.go)~~
* Examples for ~~hex strings~~ / ~~pure []byte~~
* Additional SubRecordType: EGTS_SR_ACCEL_DATA

## Installation
```go
go get -u github.com/LdDl/go-egts/...
```

## Testing
```go
go test -timeout 30s github.com/LdDl/go-egts/egts/packet
```

## Current usage
You can start TCP server and check how it is parsing EGTS data by command below (from package folder)
```go
go run example/main.go
```

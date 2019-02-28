# WORK IN PROGRESS: go-egts
EGTS (Era Glonass Telematics Standard) parsing via Golang

## PR's are welcome!

## TODO:
* Refactor code to make it more clear
* Test coverage
* ~~Example of TCP server (see /example/main.go)~~
* Examples for binary files / ~~hex strings~~ / ~~pure []byte~~
* Additional SubRecotdType

## Installation
```go
go get -u github.com/LdDl/go-egts
```

## Testing
```go
go test -timeout 30s github.com/LdDl/go-egts/egts/paсket
```

## Current usage
You can start TCP server and check how it is parsing EGTS data by command below (from package folder)
```go
go run example/main.go
```
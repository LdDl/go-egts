# go-egts [![GoDoc](https://godoc.org/github.com/LdDl/go-egts?status.svg)](https://godoc.org/github.com/LdDl/go-egts) [![Sourcegraph](https://sourcegraph.com/github.com/LdDl/go-egts/-/badge.svg)](https://sourcegraph.com/github.com/LdDl/go-egts?badge) [![Go Report Card](https://goreportcard.com/badge/github.com/LdDl/go-egts)](https://goreportcard.com/report/github.com/LdDl/go-egts) [![GitHub tag](https://img.shields.io/github/tag/LdDl/go-egts.svg)](https://github.com/LdDl/go-egts/releases) [![Build Status](https://travis-ci.com/LdDl/go-egts.svg?branch=master)](https://travis-ci.com/LdDl/go-egts)
EGTS (Era Glonass Telematics Standard) parsing via Golang

## Table of Contents

- [About](#about)
- [Installation](#installation)
- [Usage](#usage)
- [Tests](#tests)
- [Support](#support)
- [License](#license)

## About
This package provides parser for EGTS packets.

What is EGTS? This abbreviation stand for Era Glonass Telematics Standard.

This is standart protocol (over TCP) for Russian global navigation system. You can read description about it here: https://docs.cntd.ru/document/1200095098 (it's on Russian obviously).

You can check [docs_rus](/docs_rus) folder for deailed workflow of packet sample.

__PR's are welcome!__


## Installation
Simple.
```shell
go get github.com/LdDl/go-egts
```


## Usage
See [cmd](/cmd) directory of this library for examples.

* Start server
    ```shell
    go run cmd/egts_server/main.go
    ```

* Start client
    ```shell
    go run cmd/egts_client/main.go
    ```

After you start both server and client, you should see something like this
* client side
    ```shell
    go run egts_client/main.go
    2021/12/16 21:18:15 Response code: {0 0      0 0 0 0 0 0 0 0 0 <nil> 0 0}
    2021/12/16 21:18:15 Packet: {1 0 00 11 0 00 0 11 0 16 1 0 0 0 0 245 0xc0000040c0 4587 0}
    ```

* server side
    ```shell
    go run egts_server/main.go
    2021/12/16 21:18:08 Accept connection on port 8081
    2021/12/16 21:18:15 Calling handleConnection for remote address: [::1]:50840
    2021/12/16 21:18:15 PosData is:
            OID: 825791382 | Longitude: 48.362186 | Latitude: 54.287315 | Time: 2021-12-16 21:18:15.1412741 +0300 MSK m=+7.036938801
    2021/12/16 21:18:15 Result code has been sent to '[::1]:50840'
    ```

## Tests
Run following command for testing library:
```shell
go test ./egts/packet_test/
go test ./egts/subrecord_test/
```

## Support
If you have troubles or questions please [open an issue](https://github.com/LdDl/go-egts/issues/new/choose).

## License
It's MIT. You can check it [here](https://github.com/LdDl/go-egts/blob/master/LICENSE.md)

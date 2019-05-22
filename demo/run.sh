#!/bin/sh

set -eu

printf "\x1b[1m(1) build our demo code and obtain a coverage.\x1b[0m\n"
cd src
go test -cover -coverprofile=coverage.txt
cd ..

printf "\x1b[1m(2) vendor the fuzzer dependencies.\x1b[0m\n"
rm -rf fuzz/src
cd fuzz/fuzz
go mod vendor
mv vendor ../src
cd ../..

printf "\x1b[1m(3) process our vendored module with Skylight.\x1b[0m\n"
cd ../src
go build skylight.go
cd ../demo
../src/skylight -c src/coverage.txt -i src -m skylight.example/rpn -o fuzz/src/skylight.example/rpn

printf "\x1b[1m(4) build the fuzzer.\x1b[0m\n"
GO_FUZZ_BUILD="$GOPATH/bin/go-fuzz-build"
cd fuzz/fuzz
GOPATH="$(realpath ..)" "$GO_FUZZ_BUILD" -func FuzzEvaluate
cd ../..

printf "\x1b[1m(5) run the fuzzer (Ctrl+C anytime).\x1b[0m\n"
"$GOPATH/bin/go-fuzz" -bin fuzz/fuzz/gofuzzdep-fuzz.zip -func FuzzEvaluate -workdir fuzz/out/rpn

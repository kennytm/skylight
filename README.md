Skylight
========

Skylight is a Go code rewriter which analyzes a coverage profile and inserts
`panic` or `println` calls before every uncovered statement. Together with a
fuzzer, it can be used to automatically generate sample inputs that touches most
part of the source code. This is particularly useful to hit a certain coverage
ratio with generated parser code, where the edge cases may not be easily found
by human.

Example usage
-------------

Suppose we want to complete coverage of a module `skylight.example/example`.

1. Obtain the coverage profile from the tests

    ```sh
    go test -cover -coverprofile=coverage.txt
    ```

2. Since go-fuzz does not support GO111MODULE yet, we need to prepare a special
    directory hierarchy for it.

    ```
    fuzz/
        fuzz/
            go.mod
            fuzzers.go
        out/
            corpus/
                corpus1.txt
                corpus2.txt
                ...
    ```

3. In `fuzz/fuzz/fuzzers.go`, write the fuzz function against the uncovered code

    ```go
    package fuzzers

    import (
        "skylight.example/example"
        // go-fuzz support
        _ "github.com/dvyukov/go-fuzz/go-fuzz-dep"
    )

    func Fuzz(data []byte) int {
        example.MustParse(data)
        return 0
    }
    ```

4. Run `go mod vendor` to vendor the dependencies. Then move the vendor
    directory upwards to simulate the $GOPATH structure.

    ```sh
    rm -rf fuzz/src
    cd fuzz/fuzz
    go mod vendor
    mv vendor ../src
    ```

    The hierarchy should now looks like

    ```
    fuzz/
        fuzz/
            go.mod
            go.sum
            fuzzers.go
        out/
            ...
        src/
            modules.txt
            github.com/
                ...
            skylight.example/
                example/
                    ...
    ```

5. Use Skylight to rewrite the vendored source.

    ```sh
    skylight \
        -c coverage.txt \
        -i /path/to/original/code \
        -m skylight.example/example \
        -o fuzz/src/skylight.example/example
    ```

6. Build the fuzzer, and begin fuzzing.

    ```sh
    cd fuzz/fuzz
    GOPATH="$(realpath ..)" go-fuzz-build
    go-fuzz -workdir ../out
    ```

7. Add test cases involving panics from uncovered lines, then restart from the
    beginning.
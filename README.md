# go-ladybug
[![Go Reference](https://pkg.go.dev/badge/github.com/LadybugDB/go-ladybug.svg)](https://pkg.go.dev/github.com/LadybugDB/go-ladybug)
[![CI](https://github.com/LadybugDB/go-ladybug/actions/workflows/go.yml/badge.svg)](https://github.com/LadybugDB/go-ladybug/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/LadybugDB/go-ladybug)](https://goreportcard.com/report/github.com/LadybugDB/go-ladybug)
[![License](https://img.shields.io/github/license/LadybugDB/go-ladybug)](LICENSE)

Official Go language binding for [LadybugDB](https://github.com/LadybugDB/ladybug). Ladybug is an embeddable property graph database management system built for query speed and scalability. For more information, please visit the [Ladybug GitHub repository](https://github.com/LadybugDB/ladybug) or the [LadybugDB website](https://ladybugdb.com).

## Installation

There are two ways to use `go-ladybug`:

### Option 1: Using `go.work`

This option applies for situations where the user wants to clone the go-ladybug repo and configure using the local module using a go.work file.

1.  Clone `go-ladybug` to your local machine:
    ```bash
    git clone https://github.com/LadybugDB/go-ladybug.git
    ```

2.  Initialize or update your `go.work` file to include both your project and the local `go-ladybug` clone:
    ```bash
    go work init ./my-project ./go-ladybug
    # OR if go.work already exists
    go work use ./go-ladybug
    ```

3.  Add a `go:generate` directive in your project. Since `go-ladybug` is now a local module, this command will run `go generate` inside the cloned directory, where files are writable:
    ```go
    //go:generate sh -c "cd $(go list -f '{{.Dir}}' -m github.com/LadybugDB/go-ladybug) && go generate ./..."
    ```

4.  Build normally:
    ```bash
    go generate ./...
    go build
    ```

### Option 2: Add the compiled libraries to your project

If you prefer not to clone the go-ladybug repo, you can download the libraries (e.g. `lib-ladybug`) at build time. You can use the `download_lbug.sh` script directly from the repository:

1.  Add a `go:generate` directive to your `main.go` or `tools.go` to download the libraries into a local folder (e.g. `lib-ladybug`). You can use the `download_lbug.sh` script directly from the repository:
    ```go
    //go:generate sh -c "curl -sL https://raw.githubusercontent.com/LadybugDB/go-ladybug/refs/heads/master/download_lbug.sh | bash -s -- -out lib-ladybug"
    ```

2.  Run generation:
    ```bash
    go generate ./...
    ```

    This will download the libraries (and header file) into `lib-ladybug/` in your project root.

3.  Configure `go-ladybug` to use the system libraries by using the `system_ladybug` build tag. You also need to tell Cgo where to find these libraries.

    You can add Cgo directives directly to your `main.go` (or any other Go file in your main package) to point to the local `lib-ladybug` directory:

    ```go
    /*
    #cgo darwin LDFLAGS: -L${SRCDIR}/lib-ladybug -Wl,-rpath,${SRCDIR}/lib-ladybug
    #cgo linux LDFLAGS: -L${SRCDIR}/lib-ladybug -Wl,-rpath,${SRCDIR}/lib-ladybug
    #cgo windows LDFLAGS: -L${SRCDIR}/lib-ladybug
    */
    import "C"

    import (
        _ "github.com/LadybugDB/go-ladybug"
    )
    ```

    Then build with the tag:

    ```bash
    go build -tags system_ladybug
    ```

    Alternatively, you can set environment variables (useful for CI scripts):

    **Linux/macOS:**
    ```bash
    export CGO_LDFLAGS="-L$(pwd)/lib-ladybug -llbug -Wl,-rpath,$(pwd)/lib-ladybug"
    go build -tags system_ladybug
    ```

    **Windows (PowerShell):**
    ```powershell
    $env:CGO_LDFLAGS="-L$PWD/lib-ladybug -llbug_shared"
    go build -tags system_ladybug
    ```

## Get started
An example project is available in the [example](example) directory.

To run the example project, you can use the following command:

```bash
cd example
go run main.go
```

## Docs
The full documentation is available at [pkg.go.dev](https://pkg.go.dev/github.com/LadybugDB/go-ladybug).

## Tests
To run the tests, you can use the following command:

```bash
go test -v
```

## Windows Support
For Cgo to properly work on Windows, MSYS2 with `UCRT64` environment is required. You can follow the instructions below to set it up:
1. Install MSYS2 from [here](https://www.msys2.org/).
2. Install Microsoft Visual C++ 2015-2022 Redistributable (x64) from [here](https://learn.microsoft.com/en-us/cpp/windows/latest-supported-vc-redist?view=msvc-170).
3. Install the required packages by running the following command in the MSYS2 terminal:
   ```bash
   pacman -S mingw-w64-ucrt-x86_64-go mingw-w64-ucrt-x86_64-gcc
   ```
4. Add the path to `lbug_shared.dll` to your `PATH` environment variable. You can do this by running the following command in the MSYS2 terminal:
   ```bash
   export PATH="$(pwd)/lib/dynamic/windows:$PATH"
   ```
   This is required to run the test cases and examples. If you are deploying your application, you can also copy the `lbug_shared.dll` file to the same directory as your executable or to a directory that is already in the `PATH`.

For an example of how to properly set up the environment, you can also refer to our CI configuration file [here](.github/workflows/go.yml).

## Contributing
We welcome contributions to go-ladybug. By contributing to go-ladybug, you agree that your contributions will be licensed under the [MIT License](LICENSE). Please read the [contributing guide](CONTRIBUTING.md) for more information.

whitenote - The whitespace kernel for Jupyter
=============================================

`whitenote` is a whitespace kernel for Jupyter.

![](jupyter.gif)

## Install

### Required libraries

Ubuntu (focal, jammy), Debian (bullseye):
```
apt install libzmq3-dev libzmq5
```

### Build and install

```
go install .
jupyter kernelspec install --name=whitenote --user ./kernel
```

## Whitespace interpreter

The whitespace interpreter (VM) is provided as the package `github.com/makiuchi-d/whitenote/wspace`.

### REPL binary

#### Build

```
go build -o bin/wspace wspace/cmd/main.go
```

#### Usage

```
wspace <file>
    Evaluate the file
wspace
    Launch an interactive interpreter
```

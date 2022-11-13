whitenote - The whitespace kernel for Jupyter
=============================================

`whitenote` is a [whitespace](https://web.archive.org/web/20150523181043/http://compsoc.dur.ac.uk/whitespace/index.php) kernel for [Jupyter](https://jupyter.org/).

![](jupyter.gif)

## Install

### Required libraries

Ubuntu (focal, jammy), Debian (bullseye):
```
apt install libzmq3-dev libzmq5
```

### Build and install

```
git clone https://github.com/Kvieta1990/whitenote.git
cd whitenote
go install .
jupyter kernelspec install --name=whitenote --user ./kernel
```

## Whitespace interpreter

The whitespace interpreter (VM) is provided in the package `github.com/makiuchi-d/whitenote/wspace`.

### REPL binary

#### Install

```
go install github.com/makiuchi-d/whitenote/wspace/cmd/wspace@latest
```

#### Usage

```
wspace <file>
    Evaluate the file
wspace
    Launch an interactive interpreter
```

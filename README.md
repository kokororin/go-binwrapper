# Golang Binary Wrapper

[![](https://img.shields.io/badge/docs-godoc-blue.svg)](https://godoc.org/github.com/kokororin/go-binwrapper)
![Build Status](https://github.com/kokororin/go-binwrapper/actions/workflows/ci.yml/badge.svg)

Inspired by and partially ported from npm package [bin-wrapper](https://github.com/kevva/bin-wrapper)

## Install

```go get -u github.com/kokororin/go-binwrapper```

## Example of usage

See complete examples in [`binwrapper_test.go`](binwrapper_test.go):


**Important note**: Many vendors don't provide binaries for some specific platforms. For instance, common linux binaries won't work on alpine linux or arm-based linux. In that case you need to have prebuilt binaries on target platform and use SkipDownload. The example above will look like:

```
bin = binwrapper.NewBinWrapper().
		SkipDownload().
		ExecPath("cwebp")
```

Now binwrapper will run *cwebp* located in **PATH**

Use Dest to specify directory with binary:

```
bin = binwrapper.NewBinWrapper().
    SkipDownload().
    Dest("/path/to/directory").
    ExecPath("cwebp")
```

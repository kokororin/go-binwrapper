package binwrapper_test

import (
	"fmt"
	"testing"

	"github.com/kokororin/go-binwrapper"
	"github.com/stretchr/testify/assert"
)

// Example of wrapping cwebp command line tool
func ExampleNewBinWrapper() {
	base := "https://storage.googleapis.com/downloads.webmproject.org/releases/webp/"

	bin := binwrapper.NewBinWrapper().
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-mac-10.12.tar.gz").
				Os("darwin")).
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-linux-x86-32.tar.gz").
				Os("linux").
				Arch("x86")).
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-linux-x86-64.tar.gz").
				Os("linux").
				Arch("x64")).
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-windows-x64.zip").
				Os("win32").
				Arch("x64").
				ExecPath("cwebp.exe")).
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-windows-x86.zip").
				Os("win32").
				Arch("x86").
				ExecPath("cwebp.exe")).
		Strip(2).
		Dest("bin/cwebp").
		ExecPath("cwebp")

	err := bin.Run("-version")

	fmt.Printf("stdout: %s\n", string(bin.StdOut()))
	fmt.Printf("stderr: %s\n", string(bin.StdErr()))
	fmt.Printf("err: %v\n", err)
}

func TestNewBinWrapperNoError(t *testing.T) {
	base := "https://storage.googleapis.com/downloads.webmproject.org/releases/webp/"

	bin := binwrapper.NewBinWrapper().
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-mac-10.12.tar.gz").
				Os("darwin")).
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-linux-x86-32.tar.gz").
				Os("linux").
				Arch("x86")).
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-linux-x86-64.tar.gz").
				Os("linux").
				Arch("x64")).
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-windows-x64.zip").
				Os("win32").
				Arch("x64")).
		Src(
			binwrapper.NewSrc().
				URL(base + "libwebp-0.6.0-windows-x86.zip").
				Os("win32").
				Arch("x86")).
		Strip(2).
		Dest("bin/cwebp").
		ExecPath("cwebp").AutoExe()

	err := bin.Run("-version")
	assert.Nil(t, err)
}

func TestNewBinWrapperError(t *testing.T) {
	bin := binwrapper.NewBinWrapper().
		ExecPath("cwebp")

	err := bin.Run("-version")
	assert.NotNil(t, err)
}

func TestNewBinWrapperNoExtract(t *testing.T) {
	base := "https://repo1.maven.org/maven2/com/google/protobuf/protoc/4.31.1/"

	bin := binwrapper.NewBinWrapper().
		Src(
			binwrapper.NewSrc().
				URL(base + "protoc-4.31.1-osx-x86_64.exe").
				Os("darwin")).
		Src(
			binwrapper.NewSrc().
				URL(base + "protoc-4.31.1-linux-x86_32.exe").
				Os("linux").
				Arch("x86")).
		Src(
			binwrapper.NewSrc().
				URL(base + "protoc-4.31.1-linux-x86_64.exe").
				Os("linux").
				Arch("x64")).
		Src(
			binwrapper.NewSrc().
				URL(base + "protoc-4.31.1-windows-x86_64.exe").
				Os("win32").
				Arch("x64")).
		Src(
			binwrapper.NewSrc().
				URL(base + "protoc-4.31.1-windows-x86_32.exe").
				Os("win32").
				Arch("x86")).
		SkipExtract().
		Dest("bin/protoc").
		ExecPath("protoc").AutoExe()

	err := bin.Run("--version")

	fmt.Printf("stdout: %s\n", string(bin.StdOut()))
	fmt.Printf("stderr: %s\n", string(bin.StdErr()))
	fmt.Printf("err: %v\n", err)
}

// Package binwrapper provides executable wrapper that makes command line tools seamlessly available as local golang dependencies.
// Inspired by and partially ported from npm package bin-wrapper: https://github.com/kevva/bin-wrapper
package binwrapper

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Src defines executable source
type Src struct {
	url      string
	os       string
	arch     string
	execPath string
}

// BinWrapper wraps executable and provides convenient methods to interact with
type BinWrapper struct {
	src         []*Src
	dest        string
	execPath    string
	strip       int
	skipExtract bool
	autoExe     bool

	stdErr       []byte
	stdOut       []byte
	stdIn        io.Reader
	stdOutWriter io.Writer

	args    []string
	env     []string
	debug   bool
	cmd     *exec.Cmd
	timeout time.Duration
}

// NewSrc creates new Src instance
func NewSrc() *Src {
	return &Src{}
}

// URL sets a url pointing to a file to download.
func (s *Src) URL(value string) *Src {
	s.url = value
	return s
}

// Os tie the source to a specific OS. Possible values are same as runtime.GOOS
func (s *Src) Os(value string) *Src {
	s.os = value
	return s
}

// Arch tie the source to a specific arch. Possible values are same as runtime.GOARCH
func (s *Src) Arch(value string) *Src {
	s.arch = value
	return s
}

// ExecPath tie the src to a specific binary file
func (s *Src) ExecPath(value string) *Src {
	s.execPath = value
	return s
}

// NewBinWrapper creates BinWrapper instance
func NewBinWrapper() *BinWrapper {
	return &BinWrapper{}
}

// Src adds a Src to BinWrapper
func (b *BinWrapper) Src(src *Src) *BinWrapper {
	b.src = append(b.src, src)
	return b
}

// Timeout sets timeout for the command. By default it's 0 (binary will run till end).
func (b *BinWrapper) Timeout(timeout time.Duration) *BinWrapper {
	b.timeout = timeout
	return b
}

// Dest accepts a path which the files will be downloaded to
func (b *BinWrapper) Dest(dest string) *BinWrapper {
	b.dest = dest
	return b
}

// ExecPath define a file to use as the binary
func (b *BinWrapper) ExecPath(execPath string) *BinWrapper {

	if b.autoExe && runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(execPath))

		if ext != ".exe" {
			execPath += ".exe"
		}
	}

	b.execPath = execPath
	return b
}

// AutoExe adds .exe extension for windows executable path
func (b *BinWrapper) AutoExe() *BinWrapper {
	b.autoExe = true
	return b.ExecPath(b.execPath)
}

// SkipDownload skips downloading a file
func (b *BinWrapper) SkipDownload() *BinWrapper {
	b.src = nil
	return b
}

// SkipExtract skips extracting a file
func (b *BinWrapper) SkipExtract() *BinWrapper {
	b.skipExtract = true
	return b
}

// Strip strips a number of leading paths from file names on extraction.
func (b *BinWrapper) Strip(value int) *BinWrapper {
	b.strip = value
	return b
}

// Arg adds command line argument to run the binary with.
func (b *BinWrapper) Arg(name string, values ...string) *BinWrapper {
	values = append([]string{name}, values...)
	b.args = append(b.args, values...)
	return b
}

// Debug enables debug output
func (b *BinWrapper) Debug() *BinWrapper {
	b.debug = true
	return b
}

// Args returns arguments were added with Arg method
func (b *BinWrapper) Args() []string {
	return b.args
}

// Path returns the full path to the binary
func (b *BinWrapper) Path() string {
	src := osFilterObj(b.src)

	if src != nil && src.execPath != "" {
		b.ExecPath(src.execPath)
	}

	if b.dest == "." {
		return b.dest + string(filepath.Separator) + b.execPath
	}

	return filepath.Join(b.dest, b.execPath)
}

// StdIn sets reader to read executable's stdin from
func (b *BinWrapper) StdIn(reader io.Reader) *BinWrapper {
	b.stdIn = reader
	return b
}

// StdOut returns the binary's stdout after Run was called
func (b *BinWrapper) StdOut() []byte {
	return b.stdOut
}

// CombinedOutput returns combined executable's stdout and stderr
func (b *BinWrapper) CombinedOutput() []byte {
	return append(b.stdOut, b.stdErr...)
}

// SetStdOut set writer to write executable's stdout
func (b *BinWrapper) SetStdOut(writer io.Writer) *BinWrapper {
	b.stdOutWriter = writer
	return b
}

// Env specifies the environment of the executable.
// If Env is nil, Run uses the current process's environment.
// Elements of env should be in form: "ENV_VARIABLE_NAME=value"
func (b *BinWrapper) Env(env []string) *BinWrapper {
	b.env = env
	return b
}

// StdErr returns the executable's stderr after Run was called
func (b *BinWrapper) StdErr() []byte {
	return b.stdErr
}

// Reset removes all arguments set with Arg method, cleans StdOut and StdErr
func (b *BinWrapper) Reset() *BinWrapper {
	b.args = []string{}
	b.stdOut = nil
	b.stdErr = nil
	b.stdIn = nil
	b.stdOutWriter = nil
	b.env = nil
	b.cmd = nil
	return b
}

// Run runs the binary with provided arg list.
// Arg list is appended to args set through Arg method
// Returns context.DeadlineExceeded in case of timeout
func (b *BinWrapper) Run(arg ...string) error {
	if b.src != nil && len(b.src) > 0 {
		err := b.findExisting()

		if err != nil {
			return err
		}
	}

	arg = append(b.args, arg...)

	if b.debug {
		fmt.Println("BinWrapper.Run: " + b.Path() + " " + strings.Join(arg, " "))
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if b.timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), b.timeout)
	} else {
		ctx = context.Background()
		cancel = func() {

		}
	}

	defer cancel()

	b.cmd = exec.CommandContext(ctx, b.Path(), arg...)

	if b.env != nil {
		b.cmd.Env = b.env
	}

	if b.stdIn != nil {
		b.cmd.Stdin = b.stdIn
	}

	var stdout io.Reader

	if b.stdOutWriter != nil {
		b.cmd.Stdout = b.stdOutWriter
	} else {
		stdout, _ = b.cmd.StdoutPipe()
	}

	stderr, _ := b.cmd.StderrPipe()

	err := b.cmd.Start()

	if err != nil {
		return err
	}

	if stdout != nil {
		b.stdOut, _ = io.ReadAll(stdout)
	}

	b.stdErr, _ = io.ReadAll(stderr)
	err = b.cmd.Wait()

	if ctx.Err() == context.DeadlineExceeded {
		return context.DeadlineExceeded
	}

	return err
}

// Kill terminates the process
func (b *BinWrapper) Kill() error {
	if b.cmd != nil && b.cmd.Process != nil {
		return b.cmd.Process.Kill()
	}

	return nil
}

func (b *BinWrapper) findExisting() error {
	_, err := os.Stat(b.Path())

	if os.IsNotExist(err) {
		fmt.Printf("%s not found. Downloading...\n", b.Path())
		return b.download()
	} else if err != nil {
		return err
	} else {
		return nil
	}
}

func (b *BinWrapper) download() error {
	src := osFilterObj(b.src)

	if src == nil {
		return errors.New("No binary found matching your system. It's probably not supported")
	}

	file, err := b.downloadFile(src.url)

	if err != nil {
		return err
	}

	if b.skipExtract {
		// Move the downloaded file to the target executable path
		targetPath := b.Path()
		err = os.Rename(file, targetPath)
		if err != nil {
			os.Remove(file) // Clean up on error
			return err
		}

		// Set executable permissions
		err = os.Chmod(targetPath, 0755)
		if err != nil {
			return err
		}

		fmt.Printf("Executable ready at %s\n", targetPath)
	} else {
		fmt.Printf("%s downloaded. Trying to extract...\n", file)

		err = b.extractFile(file)

		if err != nil {
			return err
		}

		if src.execPath != "" {
			b.ExecPath(src.execPath)
		}
	}

	return nil
}

func (b *BinWrapper) extractFile(file string) error {
	defer os.Remove(file)

	// Determine the file extension to select the appropriate unarchiving method
	switch {
	case strings.HasSuffix(file, ".zip"):
		if err := unzip(file, b.dest); err != nil {
			fmt.Printf("%s is not a valid zip archive\n", file)
			return err
		}
	case strings.HasSuffix(file, ".tar.gz") || strings.HasSuffix(file, ".tgz"):
		if err := untar(file, b.dest); err != nil {
			fmt.Printf("%s is not a valid tar.gz archive\n", file)
			return err
		}
	default:
		return fmt.Errorf("unsupported archive format: %s", file)
	}

	if b.strip == 0 {
		return nil
	}

	return b.stripDir()
}

func (b *BinWrapper) stripDir() error {
	dir := b.dest

	var dirsToRemove []string

	for i := 0; i < b.strip; i++ {
		files, err := os.ReadDir(dir)

		if err != nil {
			return err
		}

		for _, v := range files {
			if v.IsDir() {

				if dir != b.dest {
					dirsToRemove = append(dirsToRemove, dir)
				}

				dir = filepath.Join(dir, v.Name())
				break
			}
		}
	}

	files, err := os.ReadDir(dir)

	if err != nil {
		return err
	}

	for _, v := range files {
		err := os.Rename(filepath.Join(dir, v.Name()), filepath.Join(b.dest, v.Name()))

		if err != nil {
			return err
		}
	}

	for _, v := range dirsToRemove {
		os.RemoveAll(v)
	}

	return nil
}

func (b *BinWrapper) downloadFile(value string) (string, error) {

	if b.dest == "" {
		b.dest = "."
	}

	err := os.MkdirAll(b.dest, 0755)

	if err != nil {
		return "", err
	}

	fileURL, err := url.Parse(value)

	if err != nil {
		return "", err
	}

	path := fileURL.Path

	segments := strings.Split(path, "/")
	fileName := segments[len(segments)-1]
	fileName = filepath.Join(b.dest, fileName)
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)

	if err != nil {
		return "", err
	}

	defer file.Close()

	check := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err := check.Get(value)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if !(resp.StatusCode >= 200 && resp.StatusCode < 400) {
		return "", errors.New("Unable to download " + value)
	}

	_, err = io.Copy(file, resp.Body)

	return fileName, err
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Validate file path to prevent Zip Slip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		// Create directories leading up to the file
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		// Close files to prevent resource leaks
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func untar(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	var fileReader io.Reader = file

	if strings.HasSuffix(src, ".gz") {
		if fileReader, err = gzip.NewReader(file); err != nil {
			return err
		}
	}

	tarReader := tar.NewReader(fileReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fpath := filepath.Join(dest, header.Name)

		// Validate file path to prevent Tar Slip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}
			outFile, err := os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		default:
			// Handle other file types if necessary
		}
	}
	return nil
}

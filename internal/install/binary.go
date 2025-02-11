package install

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-faster/errors"
)

type Binary struct {
	URL    string
	Name   string
	SHA256 string
}

// InstallBinary installs a binary to machine.
func InstallBinary(bin Binary) error {
	fmt.Println("> Install binary", bin.Name)
	targetBinaryPath := "/usr/local/bin/" + bin.Name
	if _, err := os.Stat(targetBinaryPath); err == nil {
		fmt.Println("> Binary already exists")
		return nil
	}
	// 1. Download to tmp.
	baseName := filepath.Base(bin.URL)
	workDir, err := os.MkdirTemp("", "ki-dl-")
	defer func() {
		_ = os.RemoveAll(workDir)
	}()
	targetName := filepath.Join(workDir, baseName)
	if err != nil {
		return errors.Wrap(err, "create temp")
	}
	{
		f, err := os.Create(targetName)
		if err != nil {
			return errors.Wrap(err, "create temp")
		}
		defer func() {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}()
		fmt.Println("> Downloading", bin.URL)
		res, err := http.Get(bin.URL)
		if err != nil {
			return errors.Wrap(err, "get")
		}
		defer func() {
			_ = res.Body.Close()
		}()
		if res.StatusCode != http.StatusOK {
			return errors.Errorf("bad status: %s", res.Status)
		}
		if _, err := io.Copy(f, res.Body); err != nil {
			return errors.Wrap(err, "copy")
		}
		if err := f.Close(); err != nil {
			return errors.Wrap(err, "close")
		}
	}
	var binaryPath string
	{
		// Unpack.
		fmt.Println("> Unpacking")
		cmd := exec.Command("tar", "-xzf", baseName)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Dir = workDir
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "tar")
		}
		// Now find a binary in directory, recursively.
		err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if info.Name() == bin.Name {
				binaryPath = path
				return io.EOF
			}
			return nil
		})
		if err != nil && !errors.Is(err, io.EOF) {
			return errors.Wrap(err, "walk")
		}
		if binaryPath == "" {
			return errors.New("binary not found")
		}
	}
	{
		// 2. Check SHA256.
		fmt.Println("> Checking SHA256")
		h := sha256.New()
		f, err := os.Open(targetName)
		if err != nil {
			return errors.Wrap(err, "open")
		}
		defer func() {
			_ = f.Close()
		}()
		if _, err := io.Copy(h, f); err != nil {
			return errors.Wrap(err, "copy")
		}
		if got := fmt.Sprintf("%x", h.Sum(nil)); got != bin.SHA256 {
			return errors.Errorf("bad sha256: %s", got)
		}
		fmt.Println("> SHA256 OK")
	}
	// Install with chmod +x
	cmd := exec.Command("install", "-m", "0755", binaryPath, targetBinaryPath)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "install")
	}
	return nil
}

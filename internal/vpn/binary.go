package vpn

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func ResolveBinaryPath() (string, error) {
	// 1. Check if it's already in the system path
	if path, err := exec.LookPath("xray"); err == nil {
		return path, nil
	}

	// 2. Check if it's in the project directory
	localPath := filepath.Join("internal", "core", "xray")
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	// 3. Download it
	return downloadBinary(localPath)
}

func downloadBinary(dest string) (string, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	var xrayArch string
	switch arch {
	case "amd64":
		xrayArch = "linux-64"
	case "arm64":
		xrayArch = "linux-arm64"
	case "386":
		xrayArch = "linux-32"
	default:
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	if osName != "linux" {
		return "", fmt.Errorf("unsupported OS: %s", osName)
	}

	url := fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/latest/download/Xray-%s.zip", xrayArch)
	
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tmpZip := dest + ".zip"
	out, err := os.Create(tmpZip)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		return "", err
	}

	r, err := zip.OpenReader(tmpZip)
	if err != nil {
		return "", err
	}
	defer r.Close()

	found := false
	for _, f := range r.File {
		if f.Name == "xray" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			dstFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return "", err
			}
			_, err = io.Copy(dstFile, rc)
			dstFile.Close()
			rc.Close()
			if err != nil {
				return "", err
			}
			found = true
			break
		}
	}
	os.Remove(tmpZip)

	if !found {
		return "", fmt.Errorf("xray binary not found in zip archive")
	}

	return dest, nil
}

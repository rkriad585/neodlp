package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"neodlp/internal/config"
)

func selfUninstall() int {
	configDir, err := config.Dir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving config directory: %v\n", err)
		return 1
	}

	fmt.Println(">>> Uninstalling neodlp...")

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving binary path: %v\n", err)
		return 1
	}
	realPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exePath = realPath
	}

	// Check whether the running binary lives inside the config directory.
	binaryInside := strings.HasPrefix(exePath, configDir+string(os.PathSeparator))

	if runtime.GOOS == "windows" && binaryInside {
		// Windows cannot delete a running executable. Remove everything
		// *except* the running binary, then launch a deferred batch script
		// that waits 1s, removes the binary and the now-empty directories.
		filepath.WalkDir(configDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.EqualFold(path, exePath) {
				os.Remove(path)
			}
			return nil
		})
		// Remove the empty bin/ sub-directory (the exe is still there so
		// the rmdir below will handle it).
		binDir := filepath.Dir(exePath)
		if entries, _ := os.ReadDir(binDir); len(entries) == 1 {
			os.Remove(binDir)
		}
		fmt.Println("OK   Removed config files.")

		batContent := fmt.Sprintf("@echo off\r\ntimeout /t 1 /nobreak >nul\r\nrmdir /s /q \"%s\" 2>nul\r\necho OK   neodlp has been uninstalled.\r\ndel /f /q \"%%~f0\"\r\n", configDir)
		batPath := filepath.Join(os.TempDir(), "neodlp-uninstall.bat")
		if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating uninstall script: %v\n", err)
			fmt.Println("Please manually delete:", exePath)
			return 1
		}
		fmt.Println("OK   Uninstall script created. neodlp will be fully removed shortly.")
		exec.Command("cmd", "/C", "start", "/B", batPath).Start()
	} else {
		// Unix: RemoveAll works even with a running binary (inode lives).
		// Windows (binary outside config): RemoveAll works fine.
		if _, err := os.Stat(configDir); err == nil {
			if err := os.RemoveAll(configDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error removing config directory: %v\n", err)
				return 1
			}
			fmt.Println("OK   Removed config directory:", configDir)
		} else {
			fmt.Println("OK   No config directory found.")
		}

		if runtime.GOOS == "windows" {
			batContent := fmt.Sprintf("@echo off\r\ntimeout /t 1 /nobreak >nul\r\ndel /f /q \"%s\" 2>nul\r\necho OK   neodlp has been uninstalled.\r\ndel /f /q \"%%~f0\"\r\n", exePath)
			batPath := filepath.Join(os.TempDir(), "neodlp-uninstall.bat")
			if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating uninstall script: %v\n", err)
				fmt.Println("Please manually delete:", exePath)
				return 1
			}
			fmt.Println("OK   Uninstall script created. Binary will be deleted shortly.")
			exec.Command("cmd", "/C", "start", "/B", batPath).Start()
		} else {
			if err := os.Remove(exePath); err != nil {
				fmt.Fprintf(os.Stderr, "Error deleting binary: %v\n", err)
				fmt.Println("Please manually delete:", exePath)
				return 1
			}
			fmt.Println("OK   Deleted binary:", exePath)
		}
	}

	fmt.Println()
	fmt.Println("To remove neodlp from your PATH, edit your shell rc file")
	fmt.Println("and delete the line containing 'neostore/neodlp/bin'.")
	fmt.Println()

	if runtime.GOOS == "windows" {
		fmt.Println(`Or run:  Invoke-RestMethod -Uri "https://raw.githubusercontent.com/rkriad585/neodlp/main/installer.ps1" | Invoke-Expression -ArgumentList "--selfuninstall"`)
	} else {
		fmt.Println("Or run: curl -fsSL https://raw.githubusercontent.com/rkriad585/neodlp/main/installer.sh | sh -s -- --selfuninstall")
	}

	fmt.Println()
	fmt.Println("Restart your terminal for PATH changes to take effect.")
	return 0
}

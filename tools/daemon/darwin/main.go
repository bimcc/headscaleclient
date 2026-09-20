// Builds unmodified upstream macOS CLI/daemon sources at the pinned module version.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	arch := flag.String("arch", runtime.GOARCH, "amd64 or arm64")
	flag.Parse()
	if err := prepare(*arch); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prepare(arch string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("macOS SDK and codesign are required; use the macOS packaging workflow")
	}
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("unsupported architecture %q", arch)
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(root, "build/daemon/manifest.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	version := "v" + manifest.Version
	data, err = exec.Command("go", "mod", "download", "-json", "tailscale.com@"+version).Output()
	if err != nil {
		return fmt.Errorf("download pinned upstream: %w", err)
	}
	var module struct{ Dir, Sum, Version string }
	if err := json.Unmarshal(data, &module); err != nil {
		return err
	}
	lock, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		return err
	}
	if module.Version != version || module.Sum == "" || !strings.Contains("\n"+string(lock), "\ntailscale.com "+version+" "+module.Sum+"\n") {
		return fmt.Errorf("upstream module checksum does not match committed go.sum")
	}
	output := filepath.Join(root, "bin/daemon/darwin-"+arch)
	if err := os.MkdirAll(filepath.Join(output, "licenses"), 0o755); err != nil {
		return err
	}
	files := map[string]string{}
	for _, name := range []string{"tailscaled", "tailscale"} {
		target := filepath.Join(output, name)
		flags := "-w -s -X tailscale.com/version.longStamp=" + manifest.Version + "-headscaleclient -X tailscale.com/version.shortStamp=" + manifest.Version
		cmd := exec.Command("go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags="+flags, "-o", target, "./cmd/"+name)
		// Build from the verified upstream module, using its dependency graph.
		cmd.Dir = module.Dir
		cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH="+arch, "CGO_ENABLED=1", "MACOSX_DEPLOYMENT_TARGET=12.0", "CGO_CFLAGS=-mmacosx-version-min=12.0", "CGO_LDFLAGS=-mmacosx-version-min=12.0")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("build %s: %w", name, err)
		}
		if out, err := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", target).CombinedOutput(); err != nil {
			return fmt.Errorf("ad-hoc sign %s: %w: %s", name, err, out)
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(content)
		files[name] = hex.EncodeToString(hash[:])
	}
	license, err := os.ReadFile(filepath.Join(module.Dir, "LICENSE"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "licenses/TAILSCALE-LICENSE.txt"), license, 0o644); err != nil {
		return err
	}
	record := struct {
		SchemaVersion int               `json:"schemaVersion"`
		Version       string            `json:"version"`
		Platform      string            `json:"platform"`
		Architecture  string            `json:"architecture"`
		Source        string            `json:"source"`
		SourceGoSum   string            `json:"sourceGoSum"`
		Toolchain     string            `json:"toolchain"`
		Build         string            `json:"build"`
		Files         map[string]string `json:"files"`
	}{1, manifest.Version, "darwin", arch, "tailscale.com@" + version, module.Sum, runtime.Version(), "BIMCC build of unmodified upstream sources; ad-hoc signed, not an official Tailscale distribution", files}
	data, err = json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "provenance.json"), append(data, '\n'), 0o644)
}

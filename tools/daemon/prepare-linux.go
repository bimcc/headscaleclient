package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type daemonManifest struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Version       string                  `json:"version"`
	Linux         map[string]payloadEntry `json:"linux"`
}

type payloadEntry struct {
	URL    string            `json:"url"`
	SHA256 string            `json:"sha256"`
	Files  map[string]string `json:"files"`
}

type provenance struct {
	SchemaVersion int               `json:"schemaVersion"`
	Upstream      string            `json:"upstream"`
	Version       string            `json:"version"`
	Platform      string            `json:"platform"`
	Architecture  string            `json:"architecture"`
	Source        string            `json:"source"`
	SourceSHA256  string            `json:"sourceSha256"`
	Files         map[string]string `json:"files"`
}

func main() {
	architecture := flag.String("arch", "", "target architecture: amd64 or arm64")
	output := flag.String("output", "", "payload output directory")
	flag.Parse()
	if *architecture == "" || *output == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := prepare(*architecture, *output); err != nil {
		fmt.Fprintln(os.Stderr, "prepare Linux daemon:", err)
		os.Exit(1)
	}
}

func prepare(architecture, output string) error {
	repositoryRoot, err := findRepositoryRoot()
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(repositoryRoot, "build", "daemon", "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest daemonManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("decode %s: %w", manifestPath, err)
	}
	entry, ok := manifest.Linux[architecture]
	if !ok {
		return fmt.Errorf("no Linux daemon manifest entry exists for %q", architecture)
	}
	if err := validateEntry(entry); err != nil {
		return err
	}

	outputPath, err := filepath.Abs(filepath.Join(repositoryRoot, output))
	if err != nil {
		return err
	}
	cacheDirectory := filepath.Join(repositoryRoot, ".task", "daemon", "linux-"+architecture)
	if err := os.MkdirAll(cacheDirectory, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return err
	}
	sourceURL, _ := url.Parse(entry.URL)
	archivePath := filepath.Join(cacheDirectory, path.Base(sourceURL.Path))
	if err := ensureDownload(entry.URL, archivePath, entry.SHA256); err != nil {
		return err
	}
	files, err := extractVerifiedFiles(archivePath, outputPath, entry.Files, manifest.Version)
	if err != nil {
		return err
	}
	licenseSource := filepath.Join(repositoryRoot, "build", "daemon", "licenses", "TAILSCALE-LICENSE.txt")
	licenseData, err := os.ReadFile(licenseSource)
	if err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(outputPath, "licenses", "TAILSCALE-LICENSE.txt"), licenseData, 0o644); err != nil {
		return err
	}
	record := provenance{
		SchemaVersion: 1,
		Upstream:      "tailscale/tailscale",
		Version:       manifest.Version,
		Platform:      "linux",
		Architecture:  architecture,
		Source:        entry.URL,
		SourceSHA256:  strings.ToLower(entry.SHA256),
		Files:         files,
	}
	provenanceData, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	provenanceData = append(provenanceData, '\n')
	if err := writeAtomic(filepath.Join(outputPath, "provenance.json"), provenanceData, 0o644); err != nil {
		return err
	}
	fmt.Printf("Prepared verified Linux daemon payload at %s\n", outputPath)
	return nil
}

func findRepositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(current, "build", "daemon", "manifest.json")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("repository root containing build/daemon/manifest.json was not found")
		}
		current = parent
	}
}

func validateEntry(entry payloadEntry) error {
	parsed, err := url.Parse(entry.URL)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "pkgs.tailscale.com") {
		return fmt.Errorf("Linux daemon source must use HTTPS on pkgs.tailscale.com: %q", entry.URL)
	}
	if _, err := decodeSHA256(entry.SHA256); err != nil {
		return fmt.Errorf("invalid archive SHA-256: %w", err)
	}
	for _, name := range []string{"tailscale", "tailscaled"} {
		hash, ok := entry.Files[name]
		if !ok {
			return fmt.Errorf("Linux daemon manifest is missing %s", name)
		}
		if _, err := decodeSHA256(hash); err != nil {
			return fmt.Errorf("invalid %s SHA-256: %w", name, err)
		}
	}
	return nil
}

func ensureDownload(source, destination, expectedHash string) error {
	if actual, err := fileSHA256(destination); err == nil && strings.EqualFold(actual, expectedHash) {
		return nil
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "HeadscaleClient daemon preparer")
	response, err := (&http.Client{Timeout: 10 * time.Minute}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %s", source, response.Status)
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".download-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(temporary, hash), response.Body); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expectedHash) {
		return fmt.Errorf("archive SHA-256 mismatch: expected %s, got %s", expectedHash, actual)
	}
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(temporaryPath, destination)
}

func extractVerifiedFiles(archivePath, outputPath string, expected map[string]string, version string) (map[string]string, error) {
	archive, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	tape := tar.NewReader(gzipReader)
	verified := make(map[string]string, len(expected))
	for {
		header, err := tape.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		name := path.Base(header.Name)
		expectedHash, wanted := expected[name]
		if !wanted || header.Typeflag != tar.TypeReg {
			continue
		}
		if _, duplicate := verified[name]; duplicate {
			return nil, fmt.Errorf("archive contains more than one %s", name)
		}
		if err := extractVerifiedFile(tape, filepath.Join(outputPath, name), expectedHash); err != nil {
			return nil, fmt.Errorf("extract %s: %w", name, err)
		}
		if err := verifyTailscaleBuildInfo(filepath.Join(outputPath, name), name, version); err != nil {
			return nil, err
		}
		verified[name] = strings.ToLower(expectedHash)
	}
	if len(verified) != len(expected) {
		return nil, fmt.Errorf("archive contains %d of %d required runtime files", len(verified), len(expected))
	}
	return verified, nil
}

func verifyTailscaleBuildInfo(filePath, command, version string) error {
	info, err := buildinfo.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %s Go build info: %w", command, err)
	}
	wantCommand := "tailscale.com/cmd/" + command
	wantVersion := "v" + strings.TrimPrefix(version, "v")
	if info.Path != wantCommand || info.Main.Path != "tailscale.com" || info.Main.Version != wantVersion {
		return fmt.Errorf("unexpected %s build info: command=%q module=%q version=%q", command, info.Path, info.Main.Path, info.Main.Version)
	}
	return nil
}

func extractVerifiedFile(reader io.Reader, destination, expectedHash string) error {
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".payload-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(temporary, hash), reader); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expectedHash) {
		return fmt.Errorf("SHA-256 mismatch: expected %s, got %s", expectedHash, actual)
	}
	if err := os.Chmod(temporaryPath, 0o755); err != nil {
		return err
	}
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(temporaryPath, destination)
}

func writeAtomic(destination string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".write-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temporaryPath, mode); err != nil {
		return err
	}
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(temporaryPath, destination)
}

func fileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func decodeSHA256(value string) ([]byte, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}
	if len(decoded) != sha256.Size {
		return nil, fmt.Errorf("expected %d bytes, got %d", sha256.Size, len(decoded))
	}
	return decoded, nil
}

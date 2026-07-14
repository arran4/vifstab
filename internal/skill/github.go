package skill

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type GitHubSource struct {
	Owner string
	Repo  string
	Path  string // optional path within repo
}

// FetchTo downloads the tarball from GitHub and extracts it safely.
// Returns the commit SHA as the revision.
func (s *GitHubSource) FetchTo(destDir string) (string, error) {
	// 1. Get default branch / latest commit to track revision
	sha, err := s.getLatestCommitSHA()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	// 2. Download tarball
	tarballURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball/%s", s.Owner, s.Repo, sha)
	req, err := http.NewRequest("GET", tarballURL, nil)
	if err != nil {
		return "", err
	}

	// Add auth if we had it, but standard public repos don't need it.
	// We'll use the default client.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code fetching tarball: %d", resp.StatusCode)
	}

	// 3. Extract tarball safely
	err = s.extractTarball(resp.Body, destDir)
	if err != nil {
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}

	return sha, nil
}

func (s *GitHubSource) getLatestCommitSHA() (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", s.Owner, s.Repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return "", err
	}
	if len(commits) == 0 {
		return "", fmt.Errorf("no commits found")
	}

	return commits[0].SHA, nil
}

func (s *GitHubSource) extractTarball(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var rootDirPrefix string // The tarball has a root directory (e.g. owner-repo-sha) we need to strip

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if rootDirPrefix == "" && header.Typeflag == tar.TypeDir {
			rootDirPrefix = header.Name
			continue
		}

		// Security: prevent path traversal
		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("tarball contains invalid path traversal: %s", header.Name)
		}

		// strip the root dir prefix
		cleanName := strings.TrimPrefix(header.Name, rootDirPrefix)

		// if a specific path is requested, filter for it
		if s.Path != "" {
			if !strings.HasPrefix(cleanName, s.Path) {
				continue // skip files not in the requested path
			}
			cleanName = strings.TrimPrefix(cleanName, s.Path)
			cleanName = strings.TrimPrefix(cleanName, "/")
		}

		if cleanName == "" {
			continue
		}

		target := filepath.Join(destDir, filepath.Clean(cleanName))

		// Security: absolute path prevention (already handled by filepath.Join, but good to be careful)
		if !strings.HasPrefix(target, filepath.Clean(destDir)) {
			return fmt.Errorf("tarball contains file escaping destination: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure parent exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644) // enforce safe permissions
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink, tar.TypeLink:
			// Security: skip symlinks to prevent escaping the directory
			continue
		}
	}
	return nil
}

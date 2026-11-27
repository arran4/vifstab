package rcs

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	rcslib "github.com/arran4/golang-rcs"
)

// HistoryPath returns the archive name without opening the source or archive.
// Only /etc/fstab uses an external backup directory; other archives are adjacent.
func HistoryPath(source, backupDir string, fs FS) (string, error) {
	if fs == nil {
		fs = DefaultFS
	}
	_, path, err := historyIdentity(source, backupDir, fs)
	return path, err
}

func historyIdentity(source, backupDir string, fs FS) (identity, path string, err error) {
	if source == "" || strings.ContainsRune(source, 0) {
		return "", "", fmt.Errorf("invalid empty or NUL-containing source path")
	}
	identity = source
	if !filepath.IsAbs(identity) {
		wd, err := fs.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve source working directory: %w", err)
		}
		if !filepath.IsAbs(wd) {
			return "", "", fmt.Errorf("source working directory is not absolute")
		}
		// Never clean symlink/..: lexical cleaning can identify a different file.
		identity = joinHistoryPath(wd, source)
	}
	if identity == "/etc/fstab" {
		if backupDir == "" {
			return "", "", fmt.Errorf("no RCS backup directory was selected for /etc/fstab")
		}
		return identity, joinHistoryPath(backupDir, "fstab,v"), nil
	}
	return identity, identity + ",v", nil
}

// Preserve symlink-sensitive components in the caller-selected backup path too.
func joinHistoryPath(dir, name string) string {
	return strings.TrimRight(dir, string(filepath.Separator)) + string(filepath.Separator) + name
}

func historyParent(path string) string {
	return path[:strings.LastIndexByte(path, filepath.Separator)]
}

func archiveAuthor() rcslib.ID {
	author := os.Getenv("USER")
	for i := 0; i < len(author); i++ {
		c := author[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || i > 0 && (c >= '0' && c <= '9' || c == '-') {
			continue
		}
		return "vifstab"
	}
	if author == "" {
		return "vifstab"
	}
	return rcslib.ID(author)
}

// LegacyHistoryError refuses to guess which source owns an unlabelled legacy
// basename archive. No legacy bytes are modified or silently imported.
type LegacyHistoryError struct {
	LegacyPath  string
	HistoryPath string
}

func (e *LegacyHistoryError) Error() string {
	return fmt.Sprintf("legacy RCS history %q has no source identity; preserve and review it before starting %q (see RCS migration guidance)", e.LegacyPath, e.HistoryPath)
}

type revisionIdentity struct {
	SHA256  string `json:"sha256"`
	Size    int    `json:"size"`
	Newline bool   `json:"newline"`
}

type archiveProvenance struct {
	Version   int                         `json:"vifstab_rcs"`
	Source    string                      `json:"source_base64"`
	Revisions map[string]revisionIdentity `json:"revisions"`
}

func contentIdentity(content []byte) revisionIdentity {
	digest := sha256.Sum256(content)
	return revisionIdentity{SHA256: hex.EncodeToString(digest[:]), Size: len(content), Newline: len(content) > 0 && content[len(content)-1] == '\n'}
}

func setProvenance(f *rcslib.File, source, revision string, content []byte, provenance *archiveProvenance) error {
	if provenance == nil {
		provenance = &archiveProvenance{Version: 1, Source: base64.StdEncoding.EncodeToString([]byte(source)), Revisions: map[string]revisionIdentity{}}
	}
	provenance.Revisions[revision] = contentIdentity(content)
	data, err := json.Marshal(provenance)
	if err != nil {
		return err
	}
	f.Description = string(data)
	return nil
}

func parseProvenance(f *rcslib.File, source string) (*archiveProvenance, error) {
	var provenance archiveProvenance
	decoder := json.NewDecoder(strings.NewReader(f.Description))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&provenance); err != nil {
		return nil, fmt.Errorf("invalid or unlabelled RCS identity: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("unexpected data after RCS identity")
	}
	if provenance.Version != 1 || provenance.Source != base64.StdEncoding.EncodeToString([]byte(source)) || provenance.Revisions == nil {
		return nil, fmt.Errorf("RCS source identity/version mismatch; history was not modified")
	}
	return &provenance, nil
}

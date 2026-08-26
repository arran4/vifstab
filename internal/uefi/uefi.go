package uefi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type EFIInfo struct {
	DevicePath     string
	UUID           string
	PartUUID       string
	MountPoint     string
	IsFAT          bool
	IsESP          bool
	HasEFITree     *bool
	NVRAMRefs      *[]NVRAMRef
	BootedMatch    *bool
}

type NVRAMRef struct {
	Name string
	Path string
}

type EFIProvider interface {
	Stat(name string) (fs.FileInfo, error)
	ReadFile(name string) ([]byte, error)
	ReadDir(name string) ([]fs.DirEntry, error)
}

type ActualEFIProvider struct{}

func (ActualEFIProvider) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }
func (ActualEFIProvider) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (ActualEFIProvider) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(name) }

func Analyze(api rules.API, efiProv EFIProvider, dev string, mountPoint string) (*EFIInfo, error) {
	if _, err := efiProv.Stat("/sys/firmware/efi"); os.IsNotExist(err) {
		return nil, nil
	}

	inv, err := api.GetBlockDevices()
	if err != nil {
		return nil, err
	}

	matched := inv.Resolve(dev)
	if matched == nil {
		return nil, nil
	}

	info := &EFIInfo{
		DevicePath: matched.Path,
		UUID:       matched.UUID,
		PartUUID:   matched.PartUUID,
		MountPoint: mountPoint,
	}

	if matched.FSType == "vfat" || matched.FSType == "fat" || matched.FSType == "msdos" {
		info.IsFAT = true
	}

	if strings.EqualFold(matched.PartType, "c12a7328-f81f-11d2-ba4b-00a0c93ec93b") {
		info.IsESP = true
	}

	if mountPoint != "" {
		isMountedHere := false
		for _, m := range matched.MountPoints {
			if m == mountPoint {
				isMountedHere = true
				break
			}
		}

		if isMountedHere {
			hasTree := false
			if _, err := efiProv.Stat(filepath.Join(mountPoint, "EFI")); err == nil {
				hasTree = true
				info.HasEFITree = &hasTree
			} else if os.IsNotExist(err) || err == fs.ErrNotExist {
				info.HasEFITree = &hasTree
			} else {
				// permission denied or other errors -> unknown
				info.HasEFITree = nil
			}
		} else {
			info.HasEFITree = nil
		}
	}

	if matched.PartUUID != "" {
		files, err := efiProv.ReadDir("/sys/firmware/efi/efivars")
		if err == nil {
			partUUIDBytes, err := parseUUID(matched.PartUUID)
			if err == nil {
				var refs []NVRAMRef
				var matchFound *bool

				allParseable := true
				hasPositiveMatch := false

				for _, file := range files {
					name := file.Name()
					if strings.HasPrefix(name, "Boot") && len(name) >= 8 && name[4:8] != "Order" && name[4:8] != "Curr" {
						data, err := efiProv.ReadFile(filepath.Join("/sys/firmware/efi/efivars", name))
						if err == nil && len(data) > 4 {
							content := data[4:]
							contains, valid := containsHardDriveMediaDevicePath(content, partUUIDBytes)
							if !valid {
								allParseable = false
							}
							if contains {
								hasPositiveMatch = true
								desc, path, _ := parseEFIPath(content)
								refs = append(refs, NVRAMRef{
									Name: desc,
									Path: path,
								})
							}
						} else {
							allParseable = false
						}
					}
				}

				if allParseable || hasPositiveMatch {
					info.NVRAMRefs = &refs
				}

				// Check BootCurrent
				currentData, err := efiProv.ReadFile("/sys/firmware/efi/efivars/BootCurrent-8be4df61-93ca-11d2-aa0d-00e098032b8c")
				if err == nil && len(currentData) >= 6 {
					currentID := binary.LittleEndian.Uint16(currentData[4:6])
					expectedBootName := fmt.Sprintf("Boot%04X", currentID)

					var currentBootParsedSuccessfully bool
					for _, file := range files {
						if strings.HasPrefix(file.Name(), expectedBootName) {
							data, err := efiProv.ReadFile(filepath.Join("/sys/firmware/efi/efivars", file.Name()))
							if err == nil && len(data) > 4 {
								content := data[4:]
								contains, valid := containsHardDriveMediaDevicePath(content, partUUIDBytes)
								if valid {
									currentBootParsedSuccessfully = true
									bVal := contains
									matchFound = &bVal
								}
							}
							break
						}
					}

					if currentBootParsedSuccessfully {
						info.BootedMatch = matchFound
					}
				}
			}
		}
	}

	return info, nil
}

func parseUUID(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return nil, fmt.Errorf("invalid UUID length")
	}

	b := make([]byte, 16)
	var d1 uint32
	var d2, d3 uint16
	var d4_0, d4_1, d4_2, d4_3, d4_4, d4_5, d4_6, d4_7 uint8

	_, err := fmt.Sscanf(s, "%08x%04x%04x%02x%02x%02x%02x%02x%02x%02x%02x",
		&d1, &d2, &d3, &d4_0, &d4_1, &d4_2, &d4_3, &d4_4, &d4_5, &d4_6, &d4_7)
	if err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint32(b[0:4], d1)
	binary.LittleEndian.PutUint16(b[4:6], d2)
	binary.LittleEndian.PutUint16(b[6:8], d3)
	b[8] = d4_0
	b[9] = d4_1
	b[10] = d4_2
	b[11] = d4_3
	b[12] = d4_4
	b[13] = d4_5
	b[14] = d4_6
	b[15] = d4_7

	return b, nil
}

func containsHardDriveMediaDevicePath(content []byte, partUUIDBytes []byte) (bool, bool) {
	if len(content) < 6 {
		return false, false
	}
	filePathListLength := binary.LittleEndian.Uint16(content[4:6])

	offset := 6
	for offset+1 < len(content) {
		u := binary.LittleEndian.Uint16(content[offset : offset+2])
		offset += 2
		if u == 0 {
			break
		}
	}

	endOfList := offset + int(filePathListLength)
	if endOfList > len(content) {
		return false, false
	}

	found := false
	isValid := false
	for offset+3 < endOfList {
		pType := content[offset]
		pSubType := content[offset+1]
		pLength := binary.LittleEndian.Uint16(content[offset+2 : offset+4])

		if pLength < 4 || offset+int(pLength) > endOfList {
			break
		}

		if pType == 0x7F && pSubType == 0xFF {
			isValid = true
			break
		}

		if pType == 0x04 && pSubType == 0x01 {
			if int(pLength) >= 42 {
				partitionFormat := content[offset+40]
				signatureType := content[offset+41]
				if partitionFormat == 0x02 && signatureType == 0x02 {
					signature := content[offset+24 : offset+40]
					if bytes.Equal(signature, partUUIDBytes) {
						found = true
					}
				}
			}
		}

		offset += int(pLength)
	}

	return found, isValid
}

func parseEFIPath(content []byte) (string, string, bool) {
	if len(content) < 6 {
		return "", "", false
	}

	filePathListLength := binary.LittleEndian.Uint16(content[4:6])

	offset := 6
	var descRunes []rune
	for offset+1 < len(content) {
		u := binary.LittleEndian.Uint16(content[offset : offset+2])
		offset += 2
		if u == 0 {
			break
		}
		descRunes = append(descRunes, rune(u))
	}
	desc := string(descRunes)

	var efiPath string

	endOfList := offset + int(filePathListLength)
	if endOfList > len(content) {
		return "", "", false
	}

	isValid := false
	for offset+3 < endOfList {
		pType := content[offset]
		pSubType := content[offset+1]
		pLength := binary.LittleEndian.Uint16(content[offset+2 : offset+4])

		if pLength < 4 || offset+int(pLength) > endOfList {
			break
		}

		if pType == 0x7F && pSubType == 0xFF {
			isValid = true
			break
		}

		if pType == 0x04 && pSubType == 0x04 {
			pathData := content[offset+4 : offset+int(pLength)]
			var pathRunes []rune
			for i := 0; i+1 < len(pathData); i += 2 {
				u := binary.LittleEndian.Uint16(pathData[i : i+2])
				if u == 0 {
					break
				}
				pathRunes = append(pathRunes, rune(u))
			}
			if efiPath != "" {
				efiPath += "\\"
			}
			efiPath += string(pathRunes)
		}

		offset += int(pLength)
	}

	return desc, efiPath, isValid
}

func FormatReport(info *EFIInfo, originalSource string) string {
	if info == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s -> %s\n", originalSource, info.DevicePath))
	sb.WriteString(fmt.Sprintf("  UUID:     %s\n", info.UUID))
	sb.WriteString(fmt.Sprintf("  PARTUUID: %s\n", info.PartUUID))

	if info.IsESP {
		sb.WriteString("  GPT role: EFI System Partition\n")
	} else {
		sb.WriteString("  GPT role: Not ESP\n")
	}

	if info.HasEFITree == nil {
		sb.WriteString("  EFI tree: unknown\n")
	} else if *info.HasEFITree {
		sb.WriteString("  EFI tree: present\n")
	} else {
		sb.WriteString("  EFI tree: absent\n")
	}

	sb.WriteString("  NVRAM refs:\n")
	if info.NVRAMRefs == nil {
		sb.WriteString("    (unknown)\n")
	} else if len(*info.NVRAMRefs) == 0 {
		sb.WriteString("    (none)\n")
	} else {
		for _, ref := range *info.NVRAMRefs {
			sb.WriteString(fmt.Sprintf("    %-18s %s\n", ref.Name, ref.Path))
		}
	}

	if info.BootedMatch == nil {
		sb.WriteString("  current loader partition: unknown\n")
	} else if *info.BootedMatch {
		sb.WriteString("  current loader partition: match\n")
	} else {
		sb.WriteString("  current loader partition: mismatch\n")
	}

	return sb.String()
}

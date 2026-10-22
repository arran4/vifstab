#!/bin/bash
set -e

cat << 'GOEOF' > patch_fstab_mapping.go
package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	b, _ := ioutil.ReadFile("internal/txtar/txtar.go")
	s := string(b)

	// Ensure PARTLABEL and ID are processed in fstab outSpec block
	oldSpecBlock := `			} else if strings.HasPrefix(origSpec, "PARTUUID=") {
				val := strings.TrimPrefix(origSpec, "PARTUUID=")
				newVal := sanitizeStr(val, "partuuid", redactedMap)
				outSpec = "PARTUUID=" + newVal
			} else if strings.HasPrefix(origSpec, "/") {`

	newSpecBlock := `			} else if strings.HasPrefix(origSpec, "PARTUUID=") {
				val := strings.TrimPrefix(origSpec, "PARTUUID=")
				newVal := sanitizeStr(val, "partuuid", redactedMap)
				outSpec = "PARTUUID=" + newVal
			} else if strings.HasPrefix(origSpec, "PARTLABEL=") {
				val := strings.TrimPrefix(origSpec, "PARTLABEL=")
				newVal := sanitizeStr(val, "partlabel", redactedMap)
				outSpec = "PARTLABEL=" + newVal
			} else if strings.HasPrefix(origSpec, "ID=") {
				val := strings.TrimPrefix(origSpec, "ID=")
				newVal := sanitizeStr(val, "id", redactedMap)
				outSpec = "ID=" + newVal
			} else if strings.HasPrefix(origSpec, "/") {`

	s = strings.Replace(s, oldSpecBlock, newSpecBlock, 1)

	oldSpecBlock2 := `			} else if strings.HasPrefix(origSpec, "/") {
			    if strings.HasPrefix(origSpec, "/dev/") {
			        if mapped, ok := redactedMap[origSpec]; ok {
			            outSpec = mapped
			        } else {
			            val := sanitizeStr(origSpec, "dev", redactedMap)
				        outSpec = "/dev/sanitized-" + strings.Trim(val, "<>")
			        }
			    } else {
				    val := sanitizeStr(origSpec, "path", redactedMap)
				    outSpec = "/sanitized-" + strings.Trim(val, "<>")
				}
			} else if strings.HasPrefix(origSpec, "//") {`

	newSpecBlock2 := `			} else if strings.HasPrefix(origSpec, "/") {
			    if mapped, ok := redactedMap[origSpec]; ok {
			        outSpec = mapped
			    } else if strings.HasPrefix(origSpec, "/dev/disk/by-uuid/") {
			        val := strings.TrimPrefix(origSpec, "/dev/disk/by-uuid/")
			        outSpec = "/dev/disk/by-uuid/" + sanitizeStr(val, "uuid", redactedMap)
			    } else if strings.HasPrefix(origSpec, "/dev/disk/by-partuuid/") {
			        val := strings.TrimPrefix(origSpec, "/dev/disk/by-partuuid/")
			        outSpec = "/dev/disk/by-partuuid/" + sanitizeStr(val, "partuuid", redactedMap)
			    } else if strings.HasPrefix(origSpec, "/dev/disk/by-label/") {
			        val := strings.TrimPrefix(origSpec, "/dev/disk/by-label/")
			        outSpec = "/dev/disk/by-label/" + sanitizeStr(val, "label", redactedMap)
			    } else if strings.HasPrefix(origSpec, "/dev/disk/by-partlabel/") {
			        val := strings.TrimPrefix(origSpec, "/dev/disk/by-partlabel/")
			        outSpec = "/dev/disk/by-partlabel/" + sanitizeStr(val, "partlabel", redactedMap)
			    } else if strings.HasPrefix(origSpec, "/dev/disk/by-id/") || strings.HasPrefix(origSpec, "/dev/disk/by-path/") {
			        // We explicitly drop hardware tracing aliases, so just fallback map it
			        val := sanitizeStr(origSpec, "dev", redactedMap)
				    outSpec = "/dev/sanitized-" + strings.Trim(val, "<>")
			    } else if strings.HasPrefix(origSpec, "/dev/") {
			        val := sanitizeStr(origSpec, "dev", redactedMap)
				    outSpec = "/dev/sanitized-" + strings.Trim(val, "<>")
			    } else {
				    val := sanitizeStr(origSpec, "path", redactedMap)
				    outSpec = "/sanitized-" + strings.Trim(val, "<>")
				}
			} else if strings.HasPrefix(origSpec, "//") {`

	s = strings.Replace(s, oldSpecBlock2, newSpecBlock2, 1)

	// Ensure inventory device path mappings do the same
	oldInvBlock := `			    if clonedInv.Devices[i].Path != "" {
			        if strings.HasPrefix(clonedInv.Devices[i].Path, "/dev/") {
			            val := sanitizeStr(clonedInv.Devices[i].Path, "dev", redactedMap)
			            newPath := "/dev/sanitized-" + strings.Trim(val, "<>")
			            redactedMap[clonedInv.Devices[i].Path] = newPath
			            clonedInv.Devices[i].Path = newPath
			        } else {
			            val := sanitizeStr(clonedInv.Devices[i].Path, "path", redactedMap)
			            newPath := "/sanitized-" + strings.Trim(val, "<>")
			            redactedMap[clonedInv.Devices[i].Path] = newPath
			            clonedInv.Devices[i].Path = newPath
			        }
			    }`

	newInvBlock := `			    if clonedInv.Devices[i].Path != "" {
			        if mapped, ok := redactedMap[clonedInv.Devices[i].Path]; ok {
			            clonedInv.Devices[i].Path = mapped
			        } else if strings.HasPrefix(clonedInv.Devices[i].Path, "/dev/") {
			            val := sanitizeStr(clonedInv.Devices[i].Path, "dev", redactedMap)
			            newPath := "/dev/sanitized-" + strings.Trim(val, "<>")
			            redactedMap[clonedInv.Devices[i].Path] = newPath
			            clonedInv.Devices[i].Path = newPath
			        } else {
			            val := sanitizeStr(clonedInv.Devices[i].Path, "path", redactedMap)
			            newPath := "/sanitized-" + strings.Trim(val, "<>")
			            redactedMap[clonedInv.Devices[i].Path] = newPath
			            clonedInv.Devices[i].Path = newPath
			        }
			    }`

	s = strings.Replace(s, oldInvBlock, newInvBlock, 1)

	ioutil.WriteFile("internal/txtar/txtar.go", []byte(s), 0644)
}
GOEOF
go run patch_fstab_mapping.go
rm patch_fstab_mapping.go

cat << 'GOEOF' > patch_rules_api_fix.go
package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	b, _ := ioutil.ReadFile("internal/rules/api.go")
	s := string(b)

	oldBlock := `		// Fallback check the original requested path on file_exists
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + dev)); ok {
			if exists, ok := v.(bool); ok {
				if exists {
					return Exists
				}
				return DoesNotExist
			}
			if result, ok := v.(ExistState); ok {
				return result
			}
		}

		if v, ok := c.Cache.Get(CacheKey("file_exists:" + dev)); ok {
			if exists, ok := v.(bool); ok {
				if exists {
					return Exists
				}
				return DoesNotExist
			}
			if result, ok := v.(ExistState); ok {
				return result
			}
		}`

	newBlock := `		// Fallback check the original requested path on file_exists
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + dev)); ok {
			if exists, ok := v.(bool); ok {
				if exists {
					return Exists
				}
				return DoesNotExist
			}
			if result, ok := v.(ExistState); ok {
				return result
			}
		}`

	s = strings.Replace(s, oldBlock, newBlock, 1)

	ioutil.WriteFile("internal/rules/api.go", []byte(s), 0644)
}
GOEOF
go run patch_rules_api_fix.go
rm patch_rules_api_fix.go

cat << 'GOEOF' > patch_txtar_test_fix.go
package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	b, _ := ioutil.ReadFile("internal/txtar/txtar_test.go")
	s := string(b)

	oldBlock := `	nullRedacted := []byte("-- manifest.json --\n{\"version\": \"1.0\", \"type\": \"system\", \"redacted\": null}\n-- input/etc/fstab --\n")
	_, err = ParseTXTAR(nullRedacted)
	if err == nil || !strings.Contains(err.Error(), "missing required field") {
	    t.Errorf("expected null redacted field error, got %v", err)
	}`

	newBlock := `	nullRedacted := []byte("-- manifest.json --\n{\"version\": \"1.0\", \"type\": \"system\", \"redacted\": null}\n-- input/etc/fstab --\n")
	_, err = ParseTXTAR(nullRedacted)
	if err == nil || !strings.Contains(err.Error(), "missing required field") {
	    t.Errorf("expected null redacted field error, got %v", err)
	}`

	s = strings.Replace(s, oldBlock, newBlock, 1)

	// Also fix TestParseTXTARLimits nil deference on empty fstab
	oldParseLimitsBlock := `	emptyFstabData := []byte("-- manifest.json --\n{\"version\": \"1.0\", \"type\": \"system\", \"redacted\": false}\n-- input/etc/fstab --\n-- kernel --\n123\n")
	snapshot2, err := ParseTXTAR(emptyFstabData)
	if err != nil {
		t.Errorf("expected empty fstab to be valid, got %v", err)
	} else if snapshot2.Fstab == nil {
		t.Errorf("expected fstab to be present but empty")
	} else if snapshot2.Manifest == nil {
	    t.Errorf("expected manifest to be present")
	} else if snapshot2.Manifest.Redacted == nil || *snapshot2.Manifest.Redacted != false {
	    t.Errorf("expected manifest redacted to be false")
	}`

	newParseLimitsBlock := `	emptyFstabData := []byte("-- manifest.json --\n{\"version\": \"1.0\", \"type\": \"system\", \"redacted\": false}\n-- input/etc/fstab --\n-- kernel --\n123\n")
	snapshot2, err := ParseTXTAR(emptyFstabData)
	if err != nil {
		t.Fatalf("expected empty fstab to be valid, got %v", err)
	}
	if snapshot2.Fstab == nil {
		t.Errorf("expected fstab to be present but empty")
	}
	if snapshot2.Manifest == nil {
	    t.Errorf("expected manifest to be present")
	} else if snapshot2.Manifest.Redacted == nil || *snapshot2.Manifest.Redacted != false {
	    t.Errorf("expected manifest redacted to be false")
	}`
	s = strings.Replace(s, oldParseLimitsBlock, newParseLimitsBlock, 1)

	ioutil.WriteFile("internal/txtar/txtar_test.go", []byte(s), 0644)
}
GOEOF
go run patch_txtar_test_fix.go
rm patch_txtar_test_fix.go

go test -count=1 ./...

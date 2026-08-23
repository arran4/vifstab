with open("cmd/vifstab/main_test.go", "r") as f:
    c = f.read()
c = c.replace('''if tableType == "gpt" {
			args = append(args, "--table-type=gpt")
		} else if tableType == "msdos" {
			args = append(args, "--table-type=msdos")
		}''', '''//nolint:staticcheck // Ignore QF1003 for tableType switch\n\t\tif tableType == "gpt" {
			args = append(args, "--table-type=gpt")
		} else if tableType == "msdos" {
			args = append(args, "--table-type=msdos")
		}''')
c = c.replace('''tmpFile.Close()''', '''_ = tmpFile.Close()''')
c = c.replace('''defer os.Remove(tmpPath)''', '''defer func() { _ = os.Remove(tmpPath) }()''')
with open("cmd/vifstab/main_test.go", "w") as f:
    f.write(c)

with open("internal/format/format_test.go", "r") as f:
    c = f.read()
c = c.replace('''if file.Name == "input/etc/fstab" {
					initialContent = file.Data
				} else if file.Name == "expected/etc/fstab" {
					expectedContent = string(file.Data)
				}''', '''//nolint:staticcheck // Ignore QF1003 for file.Name switch\n\t\t\t\tif file.Name == "input/etc/fstab" {
					initialContent = file.Data
				} else if file.Name == "expected/etc/fstab" {
					expectedContent = string(file.Data)
				}''')
with open("internal/format/format_test.go", "w") as f:
    f.write(c)

with open("internal/rcs/rcs.go", "r") as f:
    c = f.read()
c = c.replace('''var fsLayer FS = DefaultFS''', '''var fsLayer = DefaultFS''')
c = c.replace('''if _, ok := err.(*fs.PathError); !ok && err != os.ErrNotExist {

		}''', '''if _, ok := err.(*fs.PathError); !ok && err != os.ErrNotExist {
			return fmt.Errorf("failed to check RCS file stat: %w", err)
		}''')
with open("internal/rcs/rcs.go", "w") as f:
    f.write(c)

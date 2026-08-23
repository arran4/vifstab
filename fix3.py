with open("cmd/vifstab/main.go", "r") as f:
    c = f.read()
c = c.replace('''defer in.Close()''', '''defer func() { _ = in.Close() }()''')
c = c.replace('''defer os.Remove(tempFile.Name())''', '''defer func() { _ = os.Remove(tempFile.Name()) }()''')
c = c.replace('''defer tempFile.Close()''', '''defer func() { _ = tempFile.Close() }()''')
with open("cmd/vifstab/main.go", "w") as f:
    f.write(c)

with open("cmd/vifstab/main_test.go", "r") as f:
    c = f.read()
c = c.replace('''tmpFile.Close()''', '''_ = tmpFile.Close()''')
c = c.replace('''defer os.Remove(tmpPath)''', '''defer func() { _ = os.Remove(tmpPath) }()''')
c = c.replace('''if tableType == "gpt" {
			args = append(args, "--table-type=gpt")
		} else if tableType == "msdos" {
			args = append(args, "--table-type=msdos")
		}''', '''switch tableType {
		case "gpt":
			args = append(args, "--table-type=gpt")
		case "msdos":
			args = append(args, "--table-type=msdos")
		}''')
with open("cmd/vifstab/main_test.go", "w") as f:
    f.write(c)

with open("internal/format/format.go", "r") as f:
    c = f.read()
c = c.replace('''fmt.Fprintln(w, "")''', '''_, _ = fmt.Fprintln(w, "")''')
c = c.replace('''fmt.Fprintln(w, line)''', '''_, _ = fmt.Fprintln(w, line)''')
c = c.replace('''fmt.Fprintln(w, strings.Join(formattedLine, "  "))''', '''_, _ = fmt.Fprintln(w, strings.Join(formattedLine, "  "))''')
with open("internal/format/format.go", "w") as f:
    f.write(c)

with open("internal/format/format_test.go", "r") as f:
    c = f.read()
c = c.replace('''if file.Name == "input/etc/fstab" {
					initialContent = file.Data
				} else if file.Name == "expected/etc/fstab" {
					expectedContent = string(file.Data)
				}''', '''switch file.Name {
				case "input/etc/fstab":
					initialContent = file.Data
				case "expected/etc/fstab":
					expectedContent = string(file.Data)
				}''')
with open("internal/format/format_test.go", "w") as f:
    f.write(c)

with open("internal/editor/editor.go", "r") as f:
    c = f.read()
c = c.replace('''var fsLayer FS = DefaultFS''', '''var fsLayer = DefaultFS''')
with open("internal/editor/editor.go", "w") as f:
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

with open("internal/format/format.go", "r") as f:
    c = f.read()
c = c.replace('''fmt.Fprintln(w, "")''', '''_, _ = fmt.Fprintln(w, "")''')
c = c.replace('''fmt.Fprintln(w, line)''', '''_, _ = fmt.Fprintln(w, line)''')
c = c.replace('''fmt.Fprintln(w, strings.Join(formattedLine, "  "))''', '''_, _ = fmt.Fprintln(w, strings.Join(formattedLine, "  "))''')
with open("internal/format/format.go", "w") as f:
    f.write(c)

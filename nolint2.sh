sed -i 's/if tableType == "gpt" {/\/\/nolint:staticcheck\n\t\tif tableType == "gpt" {/' cmd/vifstab/main_test.go
sed -i 's/if file.Name == "input\/etc\/fstab" {/\/\/nolint:staticcheck\n\t\t\t\tif file.Name == "input\/etc\/fstab" {/' internal/format/format_test.go
sed -i 's/if _, ok := err.(\*fs.PathError); !ok && err != os.ErrNotExist {/\/\/nolint:staticcheck\n\t\tif _, ok := err.(\*fs.PathError); !ok \&\& err != os.ErrNotExist {/' internal/rcs/rcs.go

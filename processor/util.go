package processor

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func FindFiles(fileRoot string) []string {
	var files []string
	filepath.WalkDir(fileRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Println(err.Error())
		}
		baseName := filepath.Base(path)
		if baseName[0] == '.' {
			Printfln("skipping dotfile %s", path)
			return nil
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files
}

func FindFilesWithName(fileRoot string, targetName string) []string {
	var files []string
	filepath.WalkDir(fileRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Println(err.Error())
		}
		baseName := filepath.Base(path)
		if baseName == targetName {
			files = append(files, path)
		}
		return nil
	})

	return files
}

func Printfln(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

// From https://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file/74107689#74107689
//
// Copy copies the contents of the file at srcpath to a regular file
// at dstpath. If the file named by dstpath already exists, it is
// truncated. The function does not copy the file mode, file
// permission bits, or file attributes.
func Copy(srcpath, dstpath string) (err error) {
	r, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer r.Close() // ignore error: file was opened read-only.

	w, err := os.Create(dstpath)
	if err != nil {
		return err
	}

	defer func() {
		// Report the error, if any, from Close, but do so
		// only if there isn't already an outgoing error.
		if c := w.Close(); err == nil {
			err = c
		}
	}()

	_, err = io.Copy(w, r)
	return err
}

func TrimExt(path string) (string, string) {
	ext := filepath.Ext(path)
	noExt, hasExt := strings.CutSuffix(path, ext)
	if !hasExt {
		panic("Whaaa?")
	}
	return noExt, ext

}

func SafeCutPrefix(s string, prefix string) string {
	s, hasPrefix := strings.CutPrefix(s, prefix)
	if !hasPrefix {
		panic("Whaa?")
	}
	return s
}

func ScrubPath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(fmt.Sprintf("error finding absolute path for %s: %s", err.Error()))
	}
	return filepath.Clean(absPath)
}

package processor

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/hjson/hjson-go/v4"
	"gopkg.in/yaml.v3"
)

func MakeFileLoader(siteRoot string, relativeDir string, readFileFn func(string) ([]byte, error)) FileLoader {
	baseDir := filepath.Clean(filepath.Join(siteRoot, relativeDir)) + "/"
	return FileLoader{
		baseDir:    baseDir,
		readFileFn: readFileFn,
		typesMap: map[string]func([]byte, any) error{
			".yaml": yaml.Unmarshal,
			".toml": func(fileBytes []byte, output any) error {
				_, err := toml.Decode(string(fileBytes), output)
				return err
			},
			".hjson": hjson.Unmarshal,
		},
	}
}

type FileLoader struct {
	baseDir string
	// e.g. os.ReadFile
	readFileFn func(string) ([]byte, error)
	typesMap   map[string]func([]byte, any) error
}

func (l FileLoader) BaseDir() string {
	return l.baseDir
}

func (l FileLoader) SupportsFormat(s string) bool {
	ext := filepath.Ext(s)
	_, hasFormat := l.typesMap[ext]
	return hasFormat
}

func (l FileLoader) LoadFileAsBytes(s string) ([]byte, error) {
	fullPath := filepath.Join(l.baseDir, s)
	return l.readFileFn(fullPath)
}

func (l FileLoader) LoadFile(s string, output any) error {
	fullPath := filepath.Join(l.baseDir, s)
	fileBytes, err := l.readFileFn(fullPath)
	if err != nil {
		return err
	}

	return l.DeserializeBytes(s, fileBytes, output)
}

func (l FileLoader) DeserializeBytes(s string, contentBytes []byte, output any) error {
	ext := filepath.Ext(s)

	fn, hasFormat := l.typesMap[ext]
	if !hasFormat {
		panic(fmt.Sprintf("Unknown extension on file path %q", s))
	}

	return fn(contentBytes, output)
}

func (l FileLoader) FindFilesWithName(targetName string) []string {
	matches := FindFilesWithName(l.baseDir, targetName)
	l.trimPrefixes(matches)
	return matches
}

func (l FileLoader) FindFiles() []string {
	matches := FindFiles(l.baseDir)
	l.trimPrefixes(matches)
	return matches
}

func (l FileLoader) Copy(src string, dst string) error {
	fullPath := filepath.Join(l.baseDir, src)
	return Copy(fullPath, dst)
}

func (l FileLoader) trimPrefixes(matches []string) {
	for i, _ := range matches {
		matches[i] = SafeCutPrefix(matches[i], l.baseDir)
	}
}

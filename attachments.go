package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxAttachmentBytes = 256 * 1024

// expandFileReferences is an effect interpreter: the pure update loop emits a
// prompt, then this boundary performs the filesystem reads requested with @.
func expandFileReferences(prompt string) string {
	paths := referencedPaths(prompt)
	if len(paths) == 0 {
		return prompt
	}
	root, err := filepath.Abs(".")
	if err != nil {
		return prompt
	}
	var attachments strings.Builder
	for _, name := range paths {
		path, ok := safeProjectPath(root, name)
		if !ok {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if len(content) > maxAttachmentBytes {
			content = content[:maxAttachmentBytes]
		}
		fmt.Fprintf(&attachments, "\n\n<attached_file path=%q>\n%s\n</attached_file>", filepath.ToSlash(name), content)
	}
	return prompt + attachments.String()
}

func referencedPaths(prompt string) []string {
	seen := make(map[string]bool)
	var paths []string
	fields := strings.Fields(prompt)
	for index := 0; index < len(fields); index++ {
		field := fields[index]
		if !strings.HasPrefix(field, "@") || len(field) == 1 {
			continue
		}
		if strings.HasPrefix(field, "@{") {
			for !strings.Contains(field, "}") && index+1 < len(fields) {
				index++
				field += " " + fields[index]
			}
		}
		name := strings.Trim(strings.TrimPrefix(field, "@"), "\"'`()[]{}<>,;:")
		if name != "" && !seen[name] {
			seen[name] = true
			paths = append(paths, name)
		}
	}
	return paths
}

func safeProjectPath(root, name string) (string, bool) {
	path, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return "", false
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	info, err := os.Stat(path)
	return path, err == nil && !info.IsDir()
}

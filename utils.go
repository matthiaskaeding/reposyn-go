package main

import (
	"fmt"
	"strings"
)

type FileJob struct {
	path    string
	relPath string
}

func makeTag(tagContent string, closed bool) string {
	if closed {
		return fmt.Sprintf("</%v>\n", tagContent)
	} else {
		return fmt.Sprintf("<%v>\n", tagContent)
	}
}

func writeTag(builder *strings.Builder, tagContent string, closed bool) error {
	tag := makeTag(tagContent, closed)
	_, err := builder.WriteString(tag)

	return err
}

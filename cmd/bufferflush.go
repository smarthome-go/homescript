package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
)

type tuple struct {
	hash  string
	value string
}

func writeOnce(input tuple, sourceFilename string, file *zip.Writer) error {
	thisOutputFile := fmt.Sprintf("output/%s_%s.hms", sourceFilename, input.hash)

	w1, err := file.Create(thisOutputFile)
	if err != nil {
		return err
	}

	reader := bytes.NewReader([]byte(input.value))
	if _, err := io.Copy(w1, reader); err != nil {
		return err
	}

	return nil
}

func bufferFlush(tupleChan chan tuple, sourceFilename string, file *zip.Writer, doneChan chan struct{}) {
	queue := make([]tuple, 0)

	for {
		select {
		case <-doneChan:
			for _, elem := range queue {
				if err := writeOnce(elem, sourceFilename, file); err != nil {
					panic(err.Error())
				}
			}

			return
		case j := <-tupleChan:
			queue = append(queue, j)
		default:
			if len(queue) == 0 {
				continue
			}

			for _, elem := range queue {
				if err := writeOnce(elem, sourceFilename, file); err != nil {
					panic(err.Error())
				}
			}

			queue = make([]tuple, 0)
		}
	}
}

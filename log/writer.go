package log

import (
	"errors"
	"fmt"
	"os"
	"path"
)

type RotatingFile struct {
	file        *os.File
	fileName    string
	maxBytes    int64
	backupCount int
}

func NewRotatingFile(fileName string, bytes int64, backupCount int) (*RotatingFile, error) {
	if bytes <= 0 {
		return nil, errors.New("bytes must be greater than 0")
	}
	filePath := path.Dir(fileName)
	if err := os.MkdirAll(filePath, 0666); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	rf := &RotatingFile{
		file:        file,
		fileName:    fileName,
		maxBytes:    bytes,
		backupCount: backupCount,
	}
	return rf, nil
}

func (self *RotatingFile) Write(p []byte) (n int, err error) {
	self.rotating(int64(len(p)))
	return self.file.Write(p)
}

func (self *RotatingFile) Close() error {
	return self.file.Close()
}

func (self *RotatingFile) rotating(wn int64) {
	if self.backupCount < 1 {
		return
	}
	if fileInfo, err := self.file.Stat(); err != nil {
		return
	} else if fileInfo.Size()+wn < self.maxBytes {
		return
	}

	var oldPath, newPath string
	for i := self.backupCount - 1; i > 0; i-- {
		oldPath = fmt.Sprintf("%s-%d", self.fileName, i)
		newPath = fmt.Sprintf("%s-%d", self.fileName, i+1)
		os.Rename(oldPath, newPath)
	}
	self.file.Sync()
	self.file.Close()
	newPath = self.fileName + "-1"
	os.Rename(self.fileName, newPath)
	self.file, _ = os.OpenFile(self.fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
}

package log

import (
	"log"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
)

func TestLoggerStd(t *testing.T) {

	log := New(os.Stderr, LevelInfo, log.Ldate|log.Lshortfile|log.Lmicroseconds)
	log.Error("error")
	log.Warn("warn")
	log.Info("info")
	log.Debug("debug")
}

func TestLoggerFile(t *testing.T) {
	writer, err := NewRotatingFile("test.log", 4096, 5)
	if err != nil {
		t.Fatal(err)
	}

	log := New(writer, LevelDebug, DefaultFlag)

	var wait sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wait.Add(1)
		go func() {
			n := rand.Intn(10)
			t := time.Duration(int64(n))
			time.Sleep(time.Microsecond + t)
			log.Info("info log message")
			wait.Done()
		}()
	}
	wait.Wait()
}

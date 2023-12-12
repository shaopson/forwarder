package pipe

import (
	"io"
	"sync"
)

func Join(left io.ReadWriteCloser, right io.ReadWriteCloser) (wn int64, rn int64, we error, re error) {
	var wait sync.WaitGroup
	pipe := func(to io.ReadWriteCloser, from io.ReadWriteCloser, n *int64, err *error) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()
		buf := make([]byte, 16*1024)
		*n, *err = io.CopyBuffer(to, from, buf)
	}
	wait.Add(2)
	go pipe(left, right, &wn, &we)
	go pipe(right, left, &rn, &re)
	wait.Wait()
	return
}

package main

import (
	"context"
	"crypto/rand"
	"flag"
	"io"
	"io/ioutil"
	"log"
	mrand "math/rand"
	"os"
	"runtime/trace"
	"sync"
	"time"
)

const (
	writeSize   = 1024 * 1024 * 10
	parallelism = 100
)

type Flags struct {
	Trace string
}

func (f *Flags) Bind(fs *flag.FlagSet) {
	if fs == nil {
		fs = flag.CommandLine
	}
	fs.StringVar(&f.Trace, "trace", "", "Trace file.")
}

func doSomething(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "doSomething")
	defer task.End()
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer tmp.Close()
	defer os.RemoveAll(tmp.Name())
	trace.Logf(ctx, "io", "tmpFile %q", tmp.Name())

	n, err := io.Copy(tmp, io.LimitReader(rand.Reader, writeSize))
	trace.Logf(ctx, "io", "written %d bytes", n)
	return err
}

func doSomethingElse(ctx context.Context, n int) {
	ctx, task := trace.NewTask(ctx, "doSomethingElse")
	defer task.End()
	time.Sleep(time.Duration(mrand.Intn(10)+1) * time.Millisecond)

	if n > 0 {
		doSomethingElse(ctx, n-1)
	}
}

func run() error {
	ctx := context.Background()

	var wg sync.WaitGroup

	for i := 0; i < parallelism; i++ {
		wg.Add(2)

		go func() {
			doSomethingElse(ctx, 4)
			wg.Done()
		}()

		go func() {
			doSomething(ctx)
			wg.Done()
		}()
	}
	wg.Wait()
	return nil
}

func mainE(flags Flags) error {
	if flags.Trace != "" {
		f, err := os.Create(flags.Trace)
		if err != nil {
			return err
		}
		defer f.Close()
		trace.Start(f)
		defer trace.Stop()
	}

	return run()
}

func main() {
	var flags Flags
	flags.Bind(nil)
	flag.Parse()

	if err := mainE(flags); err != nil {
		log.Fatal(err)
	}
}
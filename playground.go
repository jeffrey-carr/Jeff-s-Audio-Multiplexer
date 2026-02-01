package main

import (
	"context"
	"fmt"
	"mediacenter/shared"
	"time"

	"github.com/gen2brain/malgo"
)

const sleepTime = time.Duration(50) * time.Millisecond

func reader(ctx context.Context, buf *shared.ThreadSafeBuffer[string], out chan string, workerNum int) {
	for {
		if shared.ShouldKillCtx(ctx) {
			return
		}

		if item := buf.Read(1); len(item) > 0 {
			out <- fmt.Sprintf("[Reader %d]: Got %s (%d items in buffer)", workerNum, item[0], buf.Size())
		} else {
			out <- fmt.Sprintf("[Reader %d]: Nothing to read!", workerNum)
		}

		time.Sleep(sleepTime)
	}
}

func writer(ctx context.Context, buf *shared.ThreadSafeBuffer[string], workerNum int) {
	i := 1
	for {
		if shared.ShouldKillCtx(ctx) {
			return
		}

		err := buf.Add(fmt.Sprintf("[Writer %d]: %d", workerNum, i))
		if err != nil {
			fmt.Printf("[Writer %d] Error adding to buffer: %s\n", workerNum, err.Error())
		}

		i++
		time.Sleep(sleepTime * 2)
	}
}

func speaker(in chan string) {
	for i := range in {
		fmt.Println(i)
	}
}

func listDevices(deviceType malgo.DeviceType) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		panic(fmt.Sprintf("Error creating context: %s\n", err.Error()))
	}

	infos, err := ctx.Devices(deviceType)
	for _, info := range infos {
		fmt.Println(info.Name())
	}
}

// RunPlayground allows running code arbitrarily
func RunPlayground() {
	// testing thread safe buffer
	rootCtx := context.Background()
	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()
	buf := shared.NewThreadSafeBuffer[string](100)
	out := make(chan string)

	startDelay := 3
	fmt.Printf("Playground will start in %d seconds, press any key to stop (once it starts)\n", startDelay)
	time.Sleep(time.Duration(startDelay) * time.Second)

	go speaker(out)
	nWriters := 5
	for i := range nWriters {
		go writer(ctx, buf, i+1)
	}
	nReaders := 2
	for i := range nReaders {
		go reader(ctx, buf, out, i+1)
	}

	fmt.Scanln()
	close(out)
}

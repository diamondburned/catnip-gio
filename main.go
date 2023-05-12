package main

import (
	"context"
	"image/color"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/noriah/catnip"
	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/dsp/window"
	"libdb.so/catnip-gio/internal/catnipgio"

	_ "github.com/noriah/catnip/input/all"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		w := app.NewWindow()
		if err := run(ctx, w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
}

func run(ctx context.Context, w *app.Window) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	config := catnip.Config{
		Backend:      "pipewire",
		Device:       "spotify",
		SampleRate:   128000,
		SampleSize:   1500,
		ChannelCount: 2,
		SetupFunc: func() error {
			// TODO: output.Init with the right sampling sizes and windowing
			return nil
		},
		StartFunc: func(ctx context.Context) (context.Context, error) {
			return ctx, nil
		},
		CleanupFunc: func() error {
			return nil
		},
		Windower: window.Lanczos(),
	}

	display := catnipgio.NewDisplay(config.SampleRate, config.SampleSize)
	config.Output = display.AsOutput()

	config.Analyzer = dsp.NewAnalyzer(dsp.AnalyzerConfig{
		SampleRate: config.SampleRate,
		SampleSize: config.SampleSize,
		SquashLow:  true,
		BinMethod:  dsp.MaxSampleValue(),
	})

	config.Smoother = dsp.NewSmoother(dsp.SmootherConfig{
		SampleRate:      config.SampleRate,
		SampleSize:      config.SampleSize,
		ChannelCount:    config.ChannelCount,
		SmoothingFactor: 0.6415,
		SmoothingMethod: dsp.SmoothSimpleAverage,
	})

	wg.Add(1)
	go func() {
		defer wg.Done()

		d := float64(config.SampleSize) / config.SampleRate * 1000
		log.Printf("sample duration: %.2fms (%.0fHz)\n", d, 1000/d)

		if err := catnip.Run(&config, ctx); err != nil {
			log.Fatalln(err)
		}
	}()

	go func() {
		for range time.Tick(time.Second / 120) {
			w.Invalidate()
		}
	}()

	var ops op.Ops
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				cancel()
				return e.Err

			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)

				// make window black
				clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops)
				paint.ColorOp{Color: color.NRGBA{0, 0, 0, 255}}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)

				// draw the display
				display.Layout(gtx)

				e.Frame(gtx.Ops)
			}
		}
	}
}

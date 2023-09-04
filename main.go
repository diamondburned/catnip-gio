package main

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"github.com/noriah/catnip"
	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/dsp/window"
	"github.com/spf13/pflag"
	"libdb.so/catnip-gio/internal/catnipgio"

	_ "github.com/noriah/catnip/input/all"
)

var (
	barWidth     = 15.0
	barGap       = 5.0
	backend      = "pipewire"
	device       = "easyeffects_sink"
	sampleRate   = 128000.0
	sampleSize   = 2048
	smoothFactor = 0.5
	background   = colorFlag{0, 0, 0, 255}
)

func init() {
	pflag.Float64VarP(&barWidth, "bar-width", "w", barWidth, "width of bars")
	pflag.Float64VarP(&barGap, "bar-gap", "g", barGap, "gap between bars")
	pflag.StringVarP(&backend, "backend", "b", backend, "audio backend")
	pflag.StringVarP(&device, "device", "d", device, "audio device")
	pflag.Float64VarP(&sampleRate, "sample-rate", "r", sampleRate, "sample rate")
	pflag.IntVarP(&sampleSize, "sample-size", "s", sampleSize, "sample size")
	pflag.Float64VarP(&smoothFactor, "smooth-factor", "f", smoothFactor, "smoothing factor")
	pflag.VarP(&background, "background", "B", "background color")
}

func main() {
	pflag.Parse()

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
		Backend:      backend,
		Device:       device,
		SampleRate:   sampleRate,
		SampleSize:   sampleSize,
		ChannelCount: 1,
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
		Windower: window.Hann(),
	}

	display := catnipgio.NewDisplay(config.SampleRate, config.SampleSize)
	display.SetSizes(barWidth, barGap)
	config.Output = display.AsOutput()

	config.Analyzer = dsp.NewAnalyzer(dsp.AnalyzerConfig{
		SampleRate: config.SampleRate,
		SampleSize: config.SampleSize,
		SquashLow:  true,
		BinMethod:  dsp.SumSamples(),
	})

	config.Smoother = dsp.NewSmoother(dsp.SmootherConfig{
		SampleRate:      config.SampleRate,
		SampleSize:      config.SampleSize,
		ChannelCount:    config.ChannelCount,
		SmoothingFactor: smoothFactor,
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

	var ops op.Ops
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-display.Draw:
			w.Invalidate()

		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				cancel()
				return e.Err

			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)

				// make window black
				paint.ColorOp{Color: color.NRGBA(background)}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)

				// draw the display
				display.Layout(gtx)

				e.Frame(gtx.Ops)
			}
		}
	}
}

type colorFlag color.RGBA

func (c *colorFlag) Set(s string) error {
	if !strings.HasPrefix(s, "#") {
		return fmt.Errorf("invalid color: %q", s)
	}

	s = s[1:]
	var err error

	switch len(s) {
	case 3:
		_, err = fmt.Sscanf(s, "%1x%1x%1x", &c.R, &c.G, &c.B)
		c.R *= 17
		c.G *= 17
		c.B *= 17
		c.A = 255
	case 4:
		_, err = fmt.Sscanf(s, "%1x%1x%1x%1x", &c.R, &c.G, &c.B, &c.A)
		c.R *= 17
		c.G *= 17
		c.B *= 17
		c.A *= 17
	case 6:
		_, err = fmt.Sscanf(s, "%2x%2x%2x", &c.R, &c.G, &c.B)
		c.A = 255
	case 8:
		_, err = fmt.Sscanf(s, "%2x%2x%2x%2x", &c.R, &c.G, &c.B, &c.A)
	default:
		return fmt.Errorf("invalid hexadecimal color %q", "#"+s)
	}

	if err != nil {
		return fmt.Errorf("invalid hexadecimal color %q: %w", "#"+s, err)
	}

	return nil
}

func (c *colorFlag) String() string {
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

func (c *colorFlag) RGBA() color.RGBA {
	return color.RGBA(*c)
}

func (c *colorFlag) Type() string {
	return "color"
}

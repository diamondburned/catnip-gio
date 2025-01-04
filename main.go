package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/charmbracelet/log"
	"github.com/noriah/catnip"
	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/dsp/window"
	"github.com/noriah/catnip/input"
	"github.com/spf13/pflag"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"golang.org/x/sync/errgroup"
	"libdb.so/catnip-gio/catnipgio"
	"libdb.so/catnip-gio/internal/flags"

	_ "github.com/noriah/catnip/input/all"
)

type BinMethod string

const (
	AverageSamples BinMethod = "average"
	SumSamples     BinMethod = "sum"
	MaxSampleValue BinMethod = "max"
	MinSampleValue BinMethod = "min"
)

var (
	listAll      = false
	backend      = "pipewire"
	device       = ""
	sampleRate   = 128000.0
	sampleSize   = 2048
	smoothFactor = 0.5
	decorated    = true
	barWidth     = 15.0
	barGap       = 5.0
	background   = flags.MustParseColorNRGBA("#000000")
	barColors    = flags.NewArray(",", flags.MustParseColorNRGBA("#FFFFFF"))
	drawStyle    = flags.NewStringEnum(catnipgio.DrawSymmetricVerticalBars, catnipgio.DrawVerticalBars)
	binMethod    = flags.NewStringEnum(AverageSamples, SumSamples, MaxSampleValue, MinSampleValue)
)

func init() {
	pflag.BoolVarP(&listAll, "list-all", "l", listAll, "list all audio backends and devices")
	pflag.StringVarP(&backend, "backend", "b", backend, "audio backend")
	pflag.StringVarP(&device, "device", "d", device, "audio device")
	pflag.Float64VarP(&sampleRate, "sample-rate", "r", sampleRate, "sample rate")
	pflag.IntVarP(&sampleSize, "sample-size", "s", sampleSize, "sample size")
	pflag.Float64VarP(&smoothFactor, "smooth-factor", "f", smoothFactor, "smoothing factor")
	pflag.BoolVar(&decorated, "decorated", decorated, "enable client-side window decoration")
	pflag.Float64VarP(&barWidth, "bar-width", "w", barWidth, "width of bars")
	pflag.Float64VarP(&barGap, "bar-gap", "g", barGap, "gap between bars")
	pflag.VarP(background, "background", "B", "background color")
	pflag.VarP(barColors, "bar-color", "c", "bar color gradient")
	pflag.VarP(drawStyle, "draw-style", "S", "draw style")
	pflag.VarP(binMethod, "bin-method", "m", "binning method")
}

func main() {
	loadFlags()
	pflag.Parse()
	saveFlags()

	logger := slog.New(log.NewWithOptions(os.Stderr, log.Options{
		Level: log.DebugLevel,
	}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if listAll {
		for _, backend := range input.Backends {
			devices, err := backend.Devices()
			if err != nil {
				slog.Error(
					"cannot list devices",
					"backend", backend.Name,
					"err", err)
				continue
			}

			fmt.Printf("%s:\n", backend.Name)
			for _, device := range devices {
				fmt.Printf("  - %s\n", device)
			}
			fmt.Println()
		}
		return
	}

	win := &app.Window{}
	win.Option(app.Decorated(false))
	win.Option(app.Title("catnip-gio"))
	win.Option(app.Size(unit.Dp(1000), unit.Dp(200)))

	go func() {
		if err := run(ctx, win); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error(
				"error occured",
				"err", err)
			win.Perform(system.ActionClose)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	app.Main()
}

func run(ctx context.Context, win *app.Window) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errg, ctx := errgroup.WithContext(ctx)
	defer errg.Wait()

	display := catnipgio.NewDisplay(sampleRate, sampleSize)
	display.SetSizes(barWidth, barGap)
	display.DrawStyle = catnipgio.DrawStyle(drawStyle.Value)
	switch len(barColors.Values) {
	case 1, 2:
		display.BarColors = [2]color.NRGBA{
			barColors.At(0).NRGBA(),
			barColors.At(1 % len(barColors.Values)).NRGBA(),
		}
	default:
		return fmt.Errorf("more than 2 colors specified")
	}

	errg.Go(func() error {
		// Watch for Ctrl+C and close the window when it happens.
		<-ctx.Done()
		win.Perform(system.ActionClose)
		return nil
	})

	errg.Go(func() error {
		// NOTE: This is more efficient than e.Source.Execute(op.InvalidateCmd)
		// still, despite what the documentation says. The reason for this is
		// not sure, but it might be that display.Draw actually emits less often
		// than the calculated draw frequency.
		for range display.Draw {
			win.Invalidate()
		}
		return nil
	})

	errg.Go(func() error {
		defer cancel()

		// Close the display channel when catnip is done.
		// This will cause the draw/invalidate loop to exit.
		defer close(display.Draw)

		const channelCount = 1

		config := catnip.Config{
			Backend:      backend,
			Device:       device,
			SampleRate:   sampleRate,
			SampleSize:   sampleSize,
			ChannelCount: channelCount,
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
			Output:   display.AsOutput(),
			Analyzer: dsp.NewAnalyzer(dsp.AnalyzerConfig{
				SampleRate: sampleRate,
				SampleSize: sampleSize,
				SquashLow:  true,
				BinMethod:  dsp.SumSamples(),
			}),
			Smoother: dsp.NewSmoother(dsp.SmootherConfig{
				SampleRate:      sampleRate,
				SampleSize:      sampleSize,
				ChannelCount:    channelCount,
				SmoothingFactor: smoothFactor,
				SmoothingMethod: dsp.SmoothSimpleAverage,
			}),
		}

		sampleDurationMs := float64(sampleSize) / sampleRate * 1000
		slog.Debug(
			"initializing catnip",
			"sample_rate", config.SampleRate,
			"sample_size", config.SampleSize,
			"channel_count", config.ChannelCount,
			"sample_duration", fmt.Sprintf("%.2fms", sampleDurationMs),
			"sample_frequency", fmt.Sprintf("%.0fHz", 1000/sampleDurationMs))

		return catnip.Run(&config, ctx)
	})

	errg.Go(func() error {
		defer cancel()

		th := material.NewTheme()
		th.Bg = background.NRGBA()
		th.Fg = barColors.At(0).NRGBA()
		th.ContrastBg = color.NRGBA{0, 0, 0, 0}
		th.ContrastFg = invertColor(background.NRGBA())

		const closeButtonSize = 24
		const closeButtonMargin = 4

		var closeButton widget.Clickable
		closeIcon, _ := widget.NewIcon(icons.NavigationCancel)

		var ops op.Ops
		for {
			switch e := win.Event().(type) {
			case app.DestroyEvent:
				return e.Err

			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)

				if closeButton.Clicked(gtx) {
					win.Perform(system.ActionClose)
				}

				// make window black
				paint.ColorOp{Color: background.NRGBA()}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)

				// draw the display
				display.Layout(gtx)

				// draw the close button if requested
				if decorated {
					layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.End,
					}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(closeButtonMargin).Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints = layout.Exact(image.Pt(
									gtx.Dp(closeButtonSize+closeButtonMargin*2),
									gtx.Dp(closeButtonSize+closeButtonMargin*2),
								))
								w := material.IconButton(th, &closeButton, closeIcon, "Close")
								w.Size = closeButtonSize
								w.Inset = layout.UniformInset(closeButtonMargin)
								return w.Layout(gtx)
							},
						)
					}))
				}

				// queue up the next draw
				// e.Source.Execute(op.InvalidateCmd{
				// 	At: e.Now.Add(sampleDuration),
				// })

				e.Frame(gtx.Ops)
			}
		}
	})

	return errg.Wait()
}

const (
	configBase = "catnip-gio"
	flagFile   = "flags.json"
)

func configPath(file string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot get user config dir: %w", err)
	}

	configDir = filepath.Join(configDir, configBase)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create config dir: %w", err)
	}

	return filepath.Join(configDir, file), nil
}

func saveFlags() {
	configPath, err := configPath(flagFile)
	if err != nil {
		slog.Warn(
			"cannot get config path",
			"err", err)
		return
	}

	flags := make(map[string]string)
	pflag.VisitAll(func(f *pflag.Flag) {
		flags[f.Name] = f.Value.String()
	})

	b, err := json.MarshalIndent(flags, "", "  ")
	if err != nil {
		slog.Error(
			"cannot marshal flags",
			"err", err)
		return
	}

	if err := os.WriteFile(configPath, b, 0644); err != nil {
		slog.Error(
			"cannot save flags to file",
			"path", configPath,
			"err", err)
	}
}

func loadFlags() {
	configPath, err := configPath(flagFile)
	if err != nil {
		slog.Warn(
			"cannot get config path",
			"err", err)
		return
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		slog.Error(
			"cannot read flags",
			"path", configPath,
			"err", err)
		return
	}

	var flags map[string]string
	if err := json.Unmarshal(b, &flags); err != nil {
		slog.Error(
			"cannot unmarshal flags",
			"path", configPath,
			"err", err)
		return
	}

	for name, value := range flags {
		if err := pflag.Set(name, value); err != nil {
			slog.Error(
				"cannot restore flag",
				"name", name,
				"value", value,
				"err", err)
		}
	}
}

func invertColor(c color.NRGBA) color.NRGBA {
	return color.NRGBA{R: 255 - c.R, G: 255 - c.G, B: 255 - c.B, A: c.A}
}

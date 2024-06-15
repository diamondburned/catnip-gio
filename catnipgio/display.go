package catnipgio

import (
	"image/color"
	"math"
	"sync"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/processor"

	window "github.com/noriah/catnip/util"
)

// SilenceThreshold is the threshold below which we consider the audio
// to be silent.
const SilenceThreshold = 0.0001

// SilenceFrames is the number of frames we wait before we consider the
// audio to be silent.
const SilenceFrames = 10

// Display is a display of audio data using the Cairo vector graphics
// library.
type Display struct {
	BarColors [2]color.NRGBA
	DrawStyle DrawStyle

	Draw chan struct{}

	window *window.MovingWindow
	lock   sync.Mutex

	width      int
	height     int
	binsBuffer [][]float64
	nchannels  int
	peak       float64
	scale      float64
	silence    int
	zeroes     int
	barWidth   float64
	spaceWidth float64
	binWidth   float64
}

var _ processor.Output = (*displayOutput)(nil)

// NewDisplay creates a new display.
func NewDisplay(sampleRate float64, sampleSize int) *Display {
	windowSize := ((int(ScalingWindow * sampleRate)) / sampleSize) * 2

	d := &Display{
		Draw:      make(chan struct{}, 1),
		DrawStyle: DrawSymmetricVerticalBars,
		BarColors: [2]color.NRGBA{
			{255, 255, 255, 255},
			{255, 255, 255, 255},
		},
	}
	d.window = window.NewMovingWindow(windowSize)

	d.SetSizes(20, 4)
	return d
}

// SetSizes sets the sizes of the bars and spaces in the display.
func (d *Display) SetSizes(bar, space float64) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.barWidth = bar
	d.spaceWidth = space
	d.binWidth = bar + space
}

// AsOutput returns the Display as a processor.Output.
func (d *Display) AsOutput() processor.Output {
	return (*displayOutput)(d)
}

func (d *Display) isSilent() bool {
	return d.silence >= SilenceFrames
}

type displayOutput Display

func (d *displayOutput) isSilent() bool {
	return (*Display)(d).isSilent()
}

// Write implements processor.Output.
func (d *displayOutput) Write(bins [][]float64, nchannels int) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	nbins := (*Display)(d).bins(nchannels)
	var peak float64

	for i := 0; i < nchannels; i++ {
		for _, val := range bins[i][:nbins] {
			if val > peak {
				peak = val
			}
		}
	}

	d.peak = peak
	d.scale = 1.0
	d.nchannels = nchannels

	if d.peak < SilenceThreshold {
		if d.silence < SilenceFrames {
			d.silence++
		}
	} else if d.silence != 0 {
		d.silence = 0
	}

	if !d.isSilent() {
		// Only copy over audio data if we are not silent.
		// We know this based on the given buffer, not the local buffer that we
		// copy to.
		if len(d.binsBuffer) < len(bins) || len(d.binsBuffer[0]) < len(bins[0]) {
			// Ensure that we have enough space in the buffer.
			d.binsBuffer = input.MakeBuffers(len(bins), len(bins[0]))
		}
		input.CopyBuffers(d.binsBuffer, bins)

		if d.peak >= PeakThreshold {
			// do some scaling if we are above the PeakThreshold
			vMean, vSD := d.window.Update(d.peak)
			if t := vMean + (2.0 * vSD); t > 1.0 {
				d.scale = t
			}

			d.zeroes = 0
		} else if d.zeroes < ZeroThreshold {
			d.zeroes++
		}

		select {
		case d.Draw <- struct{}{}:
		default:
		}
	}

	return nil
}

// Bins implements processor.Output.
func (d *displayOutput) Bins(nchannels int) int {
	d.lock.Lock()
	defer d.lock.Unlock()

	return (*Display)(d).bins(nchannels)
}

func (d *Display) bins(nchannels int) int {
	return d.width / int(d.binWidth) / nchannels
}

func (d *Display) Layout(gtx layout.Context) layout.Dimensions {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.nchannels == 0 {
		// Not initialized yet.
		return layout.Dimensions{}
	}

	d.width = gtx.Constraints.Min.X
	d.height = gtx.Constraints.Min.Y

	wf := float64(d.width)
	hf := float64(d.height) - 2*d.barWidth
	xo := d.spaceWidth
	yo := d.barWidth

	bins := d.binsBuffer

	delta := 1
	scale := hf / d.scale
	nbars := d.bins(d.nchannels)

	// Round up the width so we don't draw a partial bar.
	xColMax := math.Round(wf/d.binWidth) * d.binWidth

	xBin := 0
	xCol := (d.binWidth)/2 + (wf-xColMax)/2

	if d.BarColors[0] == d.BarColors[1] {
		paint.ColorOp{
			Color: d.BarColors[0],
		}.Add(gtx.Ops)
	} else {
		paint.LinearGradientOp{
			Stop1:  f32.Pt(float32(xCol), 0),
			Stop2:  f32.Pt(float32(xCol), float32(d.height)),
			Color1: d.BarColors[0],
			Color2: d.BarColors[1],
		}.Add(gtx.Ops)
	}

	var path clip.Path
	path.Begin(gtx.Ops)

	switch d.DrawStyle {
	case DrawVerticalBars:
		for _, chBins := range bins {
			for xBin < nbars && xBin >= 0 && xCol < xColMax {
				stop := calculateBar(chBins[xBin]*scale, hf)
				drawBar(&path, xo+xCol, yo+hf, yo+stop)

				xCol += d.binWidth
				xBin += delta
			}

			delta = -delta
			xBin += delta // ensure xBin is not out of bounds first.
		}

	case DrawSymmetricVerticalBars:
		lBins := bins[0]
		rBins := bins[1%len(bins)]
		center := hf / 2

		for xBin < nbars && xCol < xColMax {
			lStop := calculateBar(lBins[xBin]*scale, center)
			rStop := calculateBar(rBins[xBin]*scale, center)
			drawBar(&path, xo+xCol, yo+lStop, yo+hf-rStop)

			xCol += d.binWidth
			xBin++
		}
	}

	stack := clip.Stroke{
		Path:  path.End(),
		Width: float32(d.barWidth),
	}.Op().Push(gtx.Ops)
	defer stack.Pop()

	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{
		Size:     gtx.Constraints.Max.Sub(gtx.Constraints.Min),
		Baseline: 0,
	}
}

func drawBar(path *clip.Path, xCol, start, stop float64) {
	path.MoveTo(f32.Pt(float32(xCol), float32(start)))
	path.LineTo(f32.Pt(float32(xCol), float32(stop)))
}

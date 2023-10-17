package main

import (
	"context"
	"fmt"
	"image"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vggio"
)

const maxDataCount int = 50

func vgLengthToGioDp(cm vg.Length, dpi float64) unit.Dp {
	return unit.Dp(float32(cm.Dots(dpi)))
}

func gioDpToVgLength(dp unit.Dp, dpi float64) vg.Length {
	pixelToDp := font.Inch.Points() / dpi
	return font.Points(float64(dp) * pixelToDp)
}

func gioDpToPixel(dp unit.Dp, dpi float64) int {
	return int(dp)
	// dp = (width in pixels * 160) / screen density
	// px = dp * dpi / 160
	return int(float64(dp) * dpi / 160)
}

func drawStatusLabel(dataRef *[]StatusMessage, th *material.Theme) func(gtx layout.Context) layout.Dimensions {
	data := *dataRef

	label := func(str string) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Label(th, unit.Sp(26), str)
			return l.Layout(gtx)
		})
	}

	return func(gtx layout.Context) layout.Dimensions {
		if len(data) > 0 {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				label(fmt.Sprintf("t: %f", data[len(data)-1].Time)),
				label(fmt.Sprintf("Kp: %f", data[len(data)-1].KP)),
				label(fmt.Sprintf("Ki: %f", data[len(data)-1].KI)),
				label(fmt.Sprintf("Kd: %f", data[len(data)-1].KD)),
				label(fmt.Sprintf("u: %f", data[len(data)-1].Output)),
				label(fmt.Sprintf("e: %f", data[len(data)-1].Error)),
			)
		} else {
			return layout.Dimensions{}
		}
	}

}

func drawGainPlot(w unit.Dp, h unit.Dp, dataRef *[]StatusMessage, dpi float64) func(gtx layout.Context) layout.Dimensions {
	p := plot.New()
	p.Title.Text = "Gain"
	p.Y.Label.Text = "Value"
	p.X.Label.Text = "Time (Sec)"

	var points [3]plotter.XYs
	for i := 0; i < len(points); i++ {
		points[i] = make(plotter.XYs, maxDataCount)
	}

	data := *dataRef
	if len(data) != 0 {
		data = data[max(len(data)-1-maxDataCount, 0) : len(data)-1]
		for j := 0; j < min(maxDataCount, len(data)); j++ {
			points[0][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].KP)}
			points[1][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].KI)}
			points[2][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].KD)}
		}
	}

	if err := plotutil.AddLinePoints(p,
		"Kp", points[0],
		"Ki", points[1],
		"Kd", points[2]); err != nil {
		GlobalLogger.WithError(err).Error("failed to draw plot")
	}

	return func(gtx layout.Context) layout.Dimensions {
		cnv := vggio.New(gtx, gioDpToVgLength(w, dpi), gioDpToVgLength(h, dpi), vggio.UseDPI(int(dpi)))
		p.Draw(draw.New(cnv))
		return layout.Dimensions{
			Size:     image.Point{X: gioDpToPixel(w, dpi), Y: gioDpToPixel(h, dpi)},
			Baseline: 0,
		}
	}
}

func drawUEPlot(w unit.Dp, h unit.Dp, dataRef *[]StatusMessage, dpi float64) func(gtx layout.Context) layout.Dimensions {
	p := plot.New()
	p.Title.Text = "U & E"
	p.Y.Label.Text = "Value"
	p.X.Label.Text = "Time (Sec)"

	var points [2]plotter.XYs
	for i := 0; i < len(points); i++ {
		points[i] = make(plotter.XYs, maxDataCount)
	}

	data := *dataRef
	if len(data) != 0 {
		data = data[max(len(data)-1-maxDataCount, 0) : len(data)-1]
		for j := 0; j < min(maxDataCount, len(data)); j++ {
			points[0][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].Output)}
			points[1][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].Error)}
		}
	}

	if err := plotutil.AddLinePoints(p,
		"u", points[0],
		"e", points[1],
	); err != nil {
		GlobalLogger.WithError(err).Error("failed to draw plot")
	}

	return func(gtx layout.Context) layout.Dimensions {
		cnv := vggio.New(gtx, gioDpToVgLength(w, dpi), gioDpToVgLength(h, dpi), vggio.UseDPI(int(dpi)))
		p.Draw(draw.New(cnv))
		return layout.Dimensions{
			Size:     image.Point{X: gioDpToPixel(w, dpi), Y: gioDpToPixel(h, dpi)},
			Baseline: 0,
		}
	}
}

func loopWindow(w, h vg.Length, dpi float64, dataRef *[]StatusMessage, ctxCancel context.CancelFunc) {
	win := app.NewWindow(
		app.Title("Telemetry"),
		app.Size(
			unit.Dp(1200),
			unit.Dp(900),
		),
	)

	th := material.NewTheme()

	go func() {
		for {
			select {
			case e := <-win.Events():
				switch e := e.(type) {
				case system.FrameEvent:
					var (
						ops op.Ops
						gtx = layout.NewContext(&ops, e)
					)
					layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(drawGainPlot(unit.Dp(900), unit.Dp(400), dataRef, dpi)),
								layout.Rigid(drawUEPlot(unit.Dp(900), unit.Dp(400), dataRef, dpi)),
							)
						}),
						layout.Rigid(drawStatusLabel(dataRef, th)),
					)
					e.Frame(gtx.Ops)
				case system.DestroyEvent:
					ctxCancel()
				}
			case <-time.Tick(50 * time.Millisecond):
				win.Invalidate()
			}
		}
	}()
}

func startGUI(ctxCancel context.CancelFunc, sig <-chan SignalMessage, stat <-chan StatusMessage) {
	const (
		w   = 20 * vg.Centimeter
		h   = 15 * vg.Centimeter
		dpi = 160
	)

	var (
		data []StatusMessage
	)

	go func(bufferSize int) {
		for {
			select {
			case _ = <-sig:
				break
			case msg := <-stat:
				data = append(data, msg)
				if len(data) > 500 {
					data = data[len(data)-21 : len(data)-1]
				}
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
	}(100)

	loopWindow(w, h, dpi, &data, ctxCancel)
	app.Main()
}

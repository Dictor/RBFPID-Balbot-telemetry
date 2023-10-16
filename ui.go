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

const maxDataCount int = 80

func loopWindow(w, h vg.Length, dpi float64, dataRef *[]StatusMessage, ctxCancel context.CancelFunc) {
	win := app.NewWindow(
		app.Title("Telemetry"),
		app.Size(
			unit.Dp(float32(w.Dots(dpi)))*1.5,
			unit.Dp(float32(h.Dots(dpi))),
		),
	)

	th := material.NewTheme()

	var points [5]plotter.XYs
	for i := 0; i < len(points); i++ {
		points[i] = make(plotter.XYs, maxDataCount)
	}

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

					p := plot.New()
					p.Title.Text = "Status"
					p.Y.Label.Text = "Value"
					p.X.Label.Text = "Time (Sec)"
					//p.X.Label.TextStyle.Font.Variant = "Mono"
					//p.Y.Label.TextStyle.Font.Variant = "Mono"

					data := *dataRef
					if len(data) != 0 {
						data = data[max(len(data)-1-maxDataCount, 0) : len(data)-1]
						for j := 0; j < min(maxDataCount, len(data)); j++ {
							points[0][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].KP)}
							points[1][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].KI)}
							points[2][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].KD)}
							points[3][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].Output)}
							points[4][j] = plotter.XY{X: float64(data[j].Time), Y: float64(data[j].Error)}
						}
					}

					if err := plotutil.AddLinePoints(p,
						"Kp", points[0],
						"Ki", points[1],
						"Kd", points[2],
						"u", points[3],
						"e", points[4]); err != nil {
						GlobalLogger.WithError(err).Error("failed to draw plot")
						continue
					}

					//p.Add(plotter.NewGrid())
					chartWidth := float32(e.Size.X) * 0.66
					chartHeight := float32(e.Size.Y)
					pixelToDp := font.Inch.Points() / dpi

					label := func(str string) layout.FlexChild {
						return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(th, unit.Sp(28), str)
							l.LineHeight = 60
							return l.Layout(gtx)
						})
					}
					trans := op.Offset(image.Pt(int(unit.Dp(chartWidth))+20, 0)).Push(gtx.Ops)

					if len(data) > 0 {
						layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							label(fmt.Sprintf("t: %f", data[len(data)-1].Time)),
							label(fmt.Sprintf("Kp: %f", data[len(data)-1].KP)),
							label(fmt.Sprintf("Ki: %f", data[len(data)-1].KI)),
							label(fmt.Sprintf("Kd: %f", data[len(data)-1].KD)),
							label(fmt.Sprintf("u: %f", data[len(data)-1].Output)),
							label(fmt.Sprintf("e: %f", data[len(data)-1].Error)),
						)
					}

					trans.Pop()

					cnv := vggio.New(gtx, font.Points(float64(chartWidth)*pixelToDp), font.Points(float64(chartHeight)*pixelToDp), vggio.UseDPI(int(dpi)))
					p.Draw(draw.New(cnv))
					e.Frame(cnv.Paint())

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
		dpi = 96
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

package main

import (
	"image"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/samber/lo"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vggio"
)

type (
	graphData struct {
		Kp, Ki, Kd, E, U, T []float32
	}
)

func loopWindow(w, h vg.Length, dpi float64, data *graphData) {
	win := app.NewWindow(
		app.Title("Telemetry"),
		app.Size(
			unit.Dp(float32(w.Dots(dpi)))*1.5,
			unit.Dp(float32(h.Dots(dpi))),
		),
	)

	th := material.NewTheme()

	const maxDataCount int = 20

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

					var (
						xyKi, xyKd, xyKp, xyU, xyE plotter.XYs
						refTarget                  []*plotter.XYs = []*plotter.XYs{&xyKi, &xyKd, &xyKp, &xyU, &xyE}
						refSource                  []*[]float32   = []*[]float32{&data.Ki, &data.Kd, &data.Kp, &data.U, &data.E}
					)

					for i := 0; i < len(refSource); i++ {
						src := *(refSource[i])
						if len(src) < 1 {
							continue
						}
						*(refTarget[i]) = lo.Map[float32, plotter.XY](src[max(len(src)-maxDataCount-1, 0):len(src)-1], func(item float32, index int) plotter.XY {
							return plotter.XY{X: float64(data.T[index]), Y: float64(item)}
						})
					}

					if err := plotutil.AddLinePoints(p,
						"Ki", xyKi,
						"Kd", xyKd,
						"Kp", xyKp,
						"u", xyU,
						"e", xyE); err != nil {
						GlobalLogger.WithError(err).Error("failed to draw plot")
						continue
					}

					//p.Add(plotter.NewGrid())
					chartWidth := float32(e.Size.X) * 0.66
					chartHeight := float32(e.Size.Y)
					pixelToDp := font.Inch.Points() / dpi

					trans := op.Offset(image.Pt(int(unit.Dp(chartWidth)), 0)).Push(gtx.Ops)
					material.Label(th, unit.Sp(32), "hello").Layout(gtx)
					trans.Pop()

					cnv := vggio.New(gtx, font.Points(float64(chartWidth)*pixelToDp), font.Points(float64(chartHeight)*pixelToDp), vggio.UseDPI(int(dpi)))
					p.Draw(draw.New(cnv))
					e.Frame(cnv.Paint())

				case system.DestroyEvent:
					return
				}
			case <-time.Tick(100 * time.Millisecond):
				win.Invalidate()
			}
		}
	}()
}

func RunGUI(sig <-chan SignalMessage, stat <-chan StatusMessage) {
	const (
		w   = 20 * vg.Centimeter
		h   = 15 * vg.Centimeter
		dpi = 96
	)

	var (
		data graphData = graphData{}
	)

	cleanArray := func(arr *[]float32, size int) {
		l := len(*arr)
		GlobalLogger.Info(l)
		if l > size {
			*arr = (*arr)[l-1-size : l-1]
		}
	}

	go func(bufferSize int) {
		for {
			select {
			case _ = <-sig:
				break
			case msg := <-stat:
				data.E = append(data.E, msg.Error)
				data.Kd = append(data.Kd, msg.KD)
				data.Ki = append(data.Ki, msg.KI)
				data.Kp = append(data.Kp, msg.KP)
				data.U = append(data.U, msg.Output)
				data.T = append(data.T, msg.Time)
			case <-time.Tick(10 * time.Second):
				cleanArray(&data.E, bufferSize)
				cleanArray(&data.Kd, bufferSize)
				cleanArray(&data.Ki, bufferSize)
				cleanArray(&data.Kp, bufferSize)
				cleanArray(&data.U, bufferSize)
				cleanArray(&data.T, bufferSize)
			default:
				time.Sleep(250 * time.Millisecond)
			}
		}
	}(100)

	loopWindow(w, h, dpi, &data)
	app.Main()
}

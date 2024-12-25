package main

import (
    "fmt"
    "image/color"
    "os"
    "time"

    "github.com/pneumaticdeath/golife"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/canvas"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/layout"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
)

type LifeSim struct {
    widget.BaseWidget
    Game                                *golife.Game
    BoxDisplayMin, BoxDisplayMax        golife.Cell
    Scale                               float32 // pixel per cell
    drawingSurface                      *fyne.Container
    CellColor                           color.Color
    BackgroundColor                     color.Color
    running                             bool
    autoZoom                            bool
    StepTime                            float64
}

func (ls *LifeSim) CreateRenderer() fyne.WidgetRenderer {
    ls.Draw()
    return widget.NewSimpleRenderer(ls.drawingSurface)
}

func NewLifeSim() *LifeSim {
    sim := &LifeSim{}
    sim.Game = golife.NewGame()
    sim.BoxDisplayMin = golife.Cell{0, 0}
    sim.BoxDisplayMax = golife.Cell{10, 10}
    sim.drawingSurface = container.NewWithoutLayout()
    sim.CellColor = color.NRGBA{R: 0, G: 0, B: 180, A: 255}
    sim.BackgroundColor = color.Black
    // sim.BackgroundColor = color.White
    sim.autoZoom = true
    sim.ExtendBaseWidget(sim)
    return sim
}

func (ls *LifeSim) MinSize() fyne.Size {
    return fyne.NewSize(150, 150)
}

func (ls *LifeSim) IsRunning() bool {
    return ls.running
}

func (ls *LifeSim) Start() {
    ls.running = true
}

func (ls *LifeSim) Stop() {
    ls.running = false
}

func (ls *LifeSim) SetAutoZoom(az bool) {
    ls.autoZoom = az
}

func (ls *LifeSim) IsAutoZoom() bool {
    return ls.autoZoom
}

func (ls *LifeSim) Resize(size fyne.Size) {
    ls.Draw()
    ls.BaseWidget.Resize(size)
}

func (ls *LifeSim) Draw() {
    ls.AutoZoom()

    windowSize := ls.drawingSurface.Size()
    if windowSize.Width == 0 || windowSize.Height == 0 {
        // fmt.Println("Can't draw on a zero_sized window")
        return
    }

    displayWidth := float32(ls.BoxDisplayMax.X - ls.BoxDisplayMin.X + 1)
    displayHeight := float32(ls.BoxDisplayMax.Y - ls.BoxDisplayMin.Y + 1)

    ls.Scale = min(windowSize.Width / displayWidth, windowSize.Height / displayHeight)

    cellSize := fyne.NewSize(ls.Scale, ls.Scale)

    displayCenter := fyne.NewPos(float32(ls.BoxDisplayMax.X + ls.BoxDisplayMin.X)/2.0, 
                                float32(ls.BoxDisplayMax.Y + ls.BoxDisplayMin.Y)/2.0)
    
    windowCenter := fyne.NewPos(windowSize.Width/2.0, windowSize.Height/2.0)

    pixels := make(map[golife.Cell]int32)
    maxDens := 1

    ls.drawingSurface.RemoveAll()
    background := canvas.NewRectangle(ls.BackgroundColor)
    background.Resize(windowSize)
    background.Move(fyne.NewPos(0,0))

    ls.drawingSurface.Add(background)

    for cell, _ := range ls.Game.Population {
        window_x := windowCenter.X + ls.Scale * (float32(cell.X) - displayCenter.X) - ls.Scale/2.0
        window_y := windowCenter.Y + ls.Scale * (float32(cell.Y) - displayCenter.Y) - ls.Scale/2.0
        cellPos := fyne.NewPos(window_x, window_y)

        if window_x >= -0.5 && window_y >= -0.5 && window_x < windowSize.Width && window_y < windowSize.Height {
            if ls.Scale < 2.0 {
                pixelPos := golife.Cell{golife.Coord(window_x), golife.Coord(window_y)}
                pixels[pixelPos] += 1
                if int(pixels[pixelPos]) > maxDens {
                    maxDens = int(pixels[pixelPos])
                }
            } else {
                cellCircle := canvas.NewCircle(ls.CellColor)
                cellCircle.Resize(cellSize)
                cellCircle.Move(cellPos)

                ls.drawingSurface.Add(cellCircle)
            }
        }
    }

    if ls.Scale < 2.0 && len(pixels) > 0 {
        for pixelPos, count := range pixels {
            density := max(float32(count)/float32(maxDens), float32(0.25))
            r, g, b, a := ls.CellColor.RGBA()
            pixelColor := color.NRGBA{R: uint8(r),
                                      G: uint8(g),
                                      B: uint8(b),
                                      A: uint8(float32(a)*density)}
            pixel := canvas.NewRectangle(pixelColor)
            pixel.Resize(fyne.NewSize(2, 2))
            pixel.Move(fyne.NewPos(float32(pixelPos.X), float32(pixelPos.Y)))
            ls.drawingSurface.Add(pixel)
        }
    }

    ls.drawingSurface.Refresh()

}

func (ls *LifeSim) SetDisplayBox(minCorner, maxCorner golife.Cell) {
    if minCorner.X > maxCorner.X {
        ls.BoxDisplayMin = golife.Cell{0, 0}
        ls.BoxDisplayMax = golife.Cell{10, 10}
    } else {
        ls.BoxDisplayMin, ls.BoxDisplayMax = minCorner, maxCorner
    }
}

func (ls *LifeSim) AutoZoom() {
    if ! ls.autoZoom {
        return 
    }

    gameBoxMin, gameBoxMax := ls.Game.Population.BoundingBox()

    if gameBoxMin.X < ls.BoxDisplayMin.X {
        ls.BoxDisplayMin.X = gameBoxMin.X
    }

    if gameBoxMin.Y < ls.BoxDisplayMin.Y {
        ls.BoxDisplayMin.Y = gameBoxMin.Y
    }

    if gameBoxMax.X > ls.BoxDisplayMax.X {
        ls.BoxDisplayMax.X = gameBoxMax.X
    }

    if gameBoxMax.Y > ls.BoxDisplayMax.Y {
        ls.BoxDisplayMax.Y = gameBoxMax.Y
    }
}

func (ls *LifeSim) ResizeToFit() {
    boxDisplayMin, boxDisplayMax := ls.Game.Population.BoundingBox()
    ls.SetDisplayBox(boxDisplayMin, boxDisplayMax)
}

type StatusBar struct {
    widget.BaseWidget
    life                *LifeSim
    GenerationDisplay   *widget.Label
    CellCountDisplay    *widget.Label
    ScaleDisplay        *widget.Label
    UpdateCadence       time.Duration
    bar                 *fyne.Container
}

func NewStatusBar(sim *LifeSim) (*StatusBar) {
    genDisp := widget.NewLabel("")
    cellCountDisp := widget.NewLabel("")
    scaleDisp := widget.NewLabel("")
    statBar := &StatusBar{life: sim, GenerationDisplay: genDisp, 
                          CellCountDisplay: cellCountDisp, ScaleDisplay: scaleDisp,
                          UpdateCadence: 20*time.Millisecond}
    statBar.bar = container.New(layout.NewHBoxLayout(), widget.NewLabel("Generation:"), statBar.GenerationDisplay,
                                layout.NewSpacer(), widget.NewLabel("Live Cells:"), statBar.CellCountDisplay,
                                layout.NewSpacer(), widget.NewLabel("Scale:"), statBar.ScaleDisplay)

    statBar.ExtendBaseWidget(statBar)

    go func() {
        for {
            statBar.Refresh()
            time.Sleep(statBar.UpdateCadence)
        }
    }()

    return statBar
}

func (statBar *StatusBar) CreateRenderer() fyne.WidgetRenderer {
    return widget.NewSimpleRenderer(statBar.bar)
}

func (statBar *StatusBar) Update() {
    statBar.GenerationDisplay.SetText(fmt.Sprintf("%d", statBar.life.Game.Generation))
    statBar.CellCountDisplay.SetText(fmt.Sprintf("%d", len(statBar.life.Game.Population)))
    statBar.ScaleDisplay.SetText(fmt.Sprintf("%.3f", statBar.life.Scale))
}

func (statBar *StatusBar) Refresh() {
    statBar.Update()
    statBar.BaseWidget.Refresh()
}

type ControlBar struct {
    widget.BaseWidget
    life                *LifeSim
    backwardStepButton  *widget.Button
    runStopButton       *widget.Button
    forwardStepButton   *widget.Button
    autoZoomCheckBox    *widget.Check
    speedSlider         *widget.Slider
    bar                 *fyne.Container
}

func NewControlBar(sim *LifeSim) *ControlBar {
    controlBar := &ControlBar{}
    controlBar.life = sim

    // red := color.NRGBA{R: 180, G: 0, B: 0, A: 255}
    blue := color.NRGBA{R: 0, G: 0, B: 180, A: 255}
    green := color.NRGBA{R: 0, G: 180, B: 0, A: 255}

    // Haven't implemented this functionality yet
    controlBar.backwardStepButton = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {})
    controlBar.backwardStepButton.Disable()

    // Stub so we can pass it as part of the button
    // action.  Will be replaced later.
    setRunStopText := func(label string, icon fyne.Resource) {}

    controlBar.runStopButton = widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), func() {
        if controlBar.life.IsRunning() {
            controlBar.life.Stop()
            setRunStopText("Run", theme.MediaPlayIcon())
            controlBar.life.CellColor = blue
            controlBar.life.Draw()
        } else {
            controlBar.life.Start()
            setRunStopText("Pause", theme.MediaPauseIcon())
            controlBar.life.CellColor = green
            go controlBar.RunGame()
        }})

    setRunStopText = func(label string, icon fyne.Resource) {
        controlBar.runStopButton.SetIcon(icon)
        controlBar.runStopButton.SetText(label)
    }

    controlBar.forwardStepButton = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
        if controlBar.life.IsRunning() {
            controlBar.life.Stop()
            controlBar.life.CellColor = blue
            controlBar.life.Refresh()
        } else {
            controlBar.StepForward()
        }
    })

    controlBar.autoZoomCheckBox = widget.NewCheck("Auto Zoom", func(checked bool) { controlBar.life.SetAutoZoom(checked) })
    controlBar.autoZoomCheckBox.SetChecked(controlBar.life.IsAutoZoom())

    controlBar.speedSlider = widget.NewSlider(1.5, 500.0)
    controlBar.speedSlider.SetValue(200.0)

    controlBar.bar = container.New(layout.NewHBoxLayout(), 
                                   controlBar.backwardStepButton, controlBar.runStopButton, controlBar.forwardStepButton, layout.NewSpacer(),
                                   controlBar.autoZoomCheckBox, layout.NewSpacer(),
                                   canvas.NewText("faster", color.Black), controlBar.speedSlider, canvas.NewText("slower", color.Black))

    controlBar.ExtendBaseWidget(controlBar)
    return controlBar
}

func (controlBar *ControlBar) CreateRenderer() fyne.WidgetRenderer {
    return widget.NewSimpleRenderer(controlBar.bar)
}

func (controlBar *ControlBar) RunGame() {
    for controlBar.life.IsRunning() {
        controlBar.StepForward()
        time.Sleep(time.Duration(controlBar.speedSlider.Value)*time.Millisecond)
    }
}

func (controlBar *ControlBar) StepForward() {
    controlBar.life.Game.Next()
    controlBar.life.Draw()
}


func main() {
    myApp := app.New()
    myWindow := myApp.NewWindow("Conway's Game of Life")

    lifeSim := NewLifeSim()

    if len(os.Args) > 1 {
        lifeSim.Game = golife.Load(os.Args[1])
    } else {
        lifeSim.Game = golife.Load("default.rle")
    }
    lifeSim.ResizeToFit()
    lifeSim.Refresh()

    controlBar := NewControlBar(lifeSim)
    statusBar := NewStatusBar(lifeSim)
    content := container.NewBorder(controlBar, statusBar, nil, nil, lifeSim)
    myWindow.Resize(fyne.NewSize(500, 500))
    myWindow.SetContent(content)

    myWindow.ShowAndRun()
}


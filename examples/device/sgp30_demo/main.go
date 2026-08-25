package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/sgp30"
	"renvo.dev/examples/device/fontcache"
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
	"renvo.dev/std/strconv"
)

type dashboard struct {
	form        forms.Form
	eco2        *forms.Label
	tvoc        *forms.Label
	eco2Bar     *forms.ProgressBar
	tvocBar     *forms.ProgressBar
	quality     *forms.Label
	status      *forms.Label
	body, title *graphics.Font
}

func (d *dashboard) label(text string, bounds graphics.Rect, title bool) *forms.Label {
	label := forms.NewLabel()
	label.SetBounds(bounds)
	if title {
		label.SetFont(d.title)
	} else {
		label.SetFont(d.body)
	}
	label.SetText(text)
	d.form.Add(&label.Control)
	return label
}

func (d *dashboard) initialize(body, title *graphics.Font) {
	d.body, d.title = body, title
	d.form.Initialize(720, 1280)
	theme := forms.LightTheme()
	theme.Accent = graphics.RGBA(20, 145, 112, 255)
	d.form.ApplyTheme(theme)

	d.label("PORT A AIR QUALITY", graphics.R(36, 32, 648, 48), true)
	d.label("M5Stack SGP30 Unit", graphics.R(36, 82, 648, 32), false)

	co2Group := forms.NewGroupBox()
	co2Group.SetBounds(graphics.R(36, 150, 648, 300))
	co2Group.SetFont(body)
	co2Group.SetText("CO2 equivalent")
	d.form.Add(&co2Group.Control)
	d.eco2 = d.label("--- ppm", graphics.R(64, 205, 590, 64), true)
	d.label("Typical fresh air starts near 400 ppm", graphics.R(64, 284, 590, 32), false)
	d.eco2Bar = forms.NewProgressBar()
	d.eco2Bar.SetBounds(graphics.R(64, 345, 592, 40))
	d.eco2Bar.SetRange(400, 2000)
	d.eco2Bar.SetValue(400)
	d.form.Add(&d.eco2Bar.Control)

	tvocGroup := forms.NewGroupBox()
	tvocGroup.SetBounds(graphics.R(36, 486, 648, 300))
	tvocGroup.SetFont(body)
	tvocGroup.SetText("Total volatile organic compounds")
	d.form.Add(&tvocGroup.Control)
	d.tvoc = d.label("--- ppb", graphics.R(64, 541, 590, 64), true)
	d.label("Lower is better", graphics.R(64, 620, 590, 32), false)
	d.tvocBar = forms.NewProgressBar()
	d.tvocBar.SetBounds(graphics.R(64, 681, 592, 40))
	d.tvocBar.SetRange(0, 600)
	d.form.Add(&d.tvocBar.Control)

	d.quality = d.label("WAITING", graphics.R(36, 846, 648, 54), true)
	d.status = d.label("Starting sensor...", graphics.R(36, 910, 648, 42), false)
	d.label("Readings update once per second", graphics.R(36, 1170, 648, 32), false)
}

func (d *dashboard) disconnected() {
	d.quality.SetText("NOT CONNECTED")
	d.quality.SetForeground(graphics.RGBA(190, 45, 45, 255))
	d.status.SetText("Connect the SGP30 Unit to Port A")
}

func (d *dashboard) update(reading sgp30.Reading, samples int) {
	d.eco2.SetText(strconv.Itoa(int(reading.ECO2)) + " ppm")
	d.tvoc.SetText(strconv.Itoa(int(reading.TVOC)) + " ppb")
	d.eco2Bar.SetValue(int(reading.ECO2))
	d.tvocBar.SetValue(int(reading.TVOC))
	if samples < 15 {
		d.quality.SetText("WARMING UP")
		d.quality.SetForeground(graphics.RGBA(185, 116, 20, 255))
		d.status.SetText("Sample " + strconv.Itoa(samples) + " of 15")
	} else if reading.ECO2 >= 1500 || reading.TVOC >= 400 {
		d.quality.SetText("VENTILATE")
		d.quality.SetForeground(graphics.RGBA(190, 45, 45, 255))
		d.status.SetText("Air quality is poor")
	} else if reading.ECO2 >= 1000 || reading.TVOC >= 150 {
		d.quality.SetText("FAIR")
		d.quality.SetForeground(graphics.RGBA(185, 116, 20, 255))
		d.status.SetText("Air quality is elevated")
	} else {
		d.quality.SetText("GOOD")
		d.quality.SetForeground(graphics.RGBA(20, 145, 112, 255))
		d.status.SetText("Air quality looks good")
	}
}

func present(d *dashboard, surface *graphics.Surface) {
	if d.form.Paint(surface) {
		if !board.PresentPortrait(surface) {
			print("TAB5 SGP30 PRESENT FAIL\n")
			for {
			}
		}
		surface.ResetDirty()
	}
}

func main() {
	print("TAB5 SGP30 MAIN\n")
	if !board.InitFramebuffer() {
		print("TAB5 SGP30 DISPLAY INIT FAIL\n")
		for {
		}
	}
	print("TAB5 SGP30 DISPLAY INIT PASS\n")
	surface := board.NewPortraitSurface()
	body, title := fontcache.Body(), fontcache.Title()
	if surface == nil || body == nil || title == nil {
		print("TAB5 SGP30 UI INIT FAIL\n")
		for {
		}
	}
	print("TAB5 SGP30 UI INIT PASS\n")
	var view dashboard
	view.initialize(body, title)
	present(&view, surface)
	print("TAB5 SGP30 PRESENT PASS\n")

	bus := i2c.New(board.PortA())
	sensor := sgp30.New(bus)
	ready, samples := false, 0
	last := board.Milliseconds() - 1000
	for {
		board.Refresh()
		now := board.Milliseconds()
		if now-last < 1000 {
			continue
		}
		last = now
		if !ready {
			if err := sensor.Initialize(); err != nil {
				view.disconnected()
				present(&view, surface)
				continue
			}
			ready = true
			print("TAB5 SGP30 SENSOR INIT PASS\n")
			view.status.SetText("Sensor ready; collecting first sample")
			present(&view, surface)
			continue
		}
		reading, err := sensor.Read()
		if err != nil {
			ready = false
			samples = 0
			view.disconnected()
			present(&view, surface)
			continue
		}
		samples++
		view.update(reading, samples)
		present(&view, surface)
	}
}

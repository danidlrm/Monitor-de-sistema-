package main

import (
	"fmt"
	"log"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("falló al iniciar termui: %v", err)
	}
	defer ui.Close()

	// Crear widgets
	cpuBar := widgets.NewGauge()
	cpuBar.Title = "CPU USO"
	cpuBar.SetRect(0, 0, 50, 3)
	cpuBar.BarColor = ui.ColorRed
	cpuBar.BorderStyle.Fg = ui.ColorWhite
	cpuBar.TitleStyle.Fg = ui.ColorCyan

	ramBar := widgets.NewGauge()
	ramBar.Title = "RAM USO"
	ramBar.SetRect(0, 4, 50, 7)
	ramBar.BarColor = ui.ColorGreen
	ramBar.BorderStyle.Fg = ui.ColorWhite
	ramBar.TitleStyle.Fg = ui.ColorCyan

	diskBar := widgets.NewGauge()
	diskBar.Title = "DISCO USO"
	diskBar.SetRect(0, 8, 50, 11)
	diskBar.BarColor = ui.ColorMagenta
	diskBar.BorderStyle.Fg = ui.ColorWhite
	diskBar.TitleStyle.Fg = ui.ColorCyan

	ui.Render(cpuBar, ramBar, diskBar)

	ticker := time.NewTicker(2 * time.Second).C

	for {
		select {
		case <-ticker:
			// CPU
			cpuPercent, _ := cpu.Percent(0, false)
			cpuBar.Percent = int(cpuPercent[0])
			cpuBar.Label = fmt.Sprintf("%.2f%%", cpuPercent[0])

			// RAM
			vm, _ := mem.VirtualMemory()
			ramBar.Percent = int(vm.UsedPercent)
			ramBar.Label = fmt.Sprintf("%.2f%% (%.2f GB de %.2f GB)", vm.UsedPercent, float64(vm.Used)/1e9, float64(vm.Total)/1e9)

			// DISCO
			ds, _ := disk.Usage("/")
			diskBar.Percent = int(ds.UsedPercent)
			diskBar.Label = fmt.Sprintf("%.2f%% (%.2f GB de %.2f GB)", ds.UsedPercent, float64(ds.Used)/1e9, float64(ds.Total)/1e9)

			ui.Render(cpuBar, ramBar, diskBar)
		case e := <-ui.PollEvents():
			if e.Type == ui.KeyboardEvent {
				if e.ID == "q" || e.ID == "<C-c>" {
					return // salir con "q" o Ctrl+C
				}
			}
		}
	}
}

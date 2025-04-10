package main

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func main() {
	for {
		// Sacamos el uso de CPU
		cpuPercent, _ := cpu.Percent(0, false)

		// Sacamos el uso de RAM
		vmStat, _ := mem.VirtualMemory()

		// Sacamos el uso del disco
		diskStat, _ := disk.Usage("/")

		// Mostramos todo bonito
		fmt.Printf("\n========== MONITOR DE SISTEMA ==========\n")
		fmt.Printf("CPU USO: %.2f%%\n", cpuPercent[0])
		fmt.Printf("RAM USO: %.2f%% (%.2f GB de %.2f GB)\n", vmStat.UsedPercent, float64(vmStat.Used)/1e9, float64(vmStat.Total)/1e9)
		fmt.Printf("DISCO USO: %.2f%% (%.2f GB de %.2f GB)\n", diskStat.UsedPercent, float64(diskStat.Used)/1e9, float64(diskStat.Total)/1e9)
		fmt.Println("========================================")

		time.Sleep(3 * time.Second)
	}
}

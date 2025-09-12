package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// Struct para guardar la info de un proceso
type Proceso struct {
	PID        int32
	Nombre     string
	CPU        float64
	RAM        float64
	TamanioRAM uint64
}

func main() {
	// Prepara la pantalla
	t, err := tcell.NewScreen()
	if err != nil || t.Init() != nil {
		fmt.Println("No se pudo iniciar la pantalla")
		os.Exit(1)
	}
	defer t.Fini()
	t.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
	t.Clear()

	// Contrl para cerrar con ctrl + c o esc
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()

	senal := make(chan os.Signal, 1)
	signal.Notify(senal, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	// Loop de actualización cada segundo
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-senal:
				cancelar()
				return
			default:
				actualizarPantalla(t)
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// Espera tecla Esc para salir
	for {
		evento := t.PollEvent()
		switch ev := evento.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape {
				cancelar()
				return
			}
		case *tcell.EventResize:
			t.Sync()
		}
	}
}

// Actualiza todos los datos en pantalla
func actualizarPantalla(t tcell.Screen) {
	t.Clear()
	ancho, alto := t.Size()

	memoria, _ := mem.VirtualMemory()
	cpuUso, _ := cpu.Percent(0, false)
	netDatos, _ := net.IOCounters(false)
	interfaces, _ := net.Interfaces()

	// Porcentaje de memoria
	usoMem := float64(memoria.Used) / float64(memoria.Total) * 100
	dibujarBarra(t, 1, 1, ancho-2, "Memoria", usoMem,
		fmt.Sprintf("Total: %.1fGB Usada: %.1fGB Libre: %.1fGB (%.1f%%)",
			float64(memoria.Total)/1e9, float64(memoria.Used)/1e9,
			float64(memoria.Free)/1e9, usoMem))

	// Uso de CPU
	usoCPU := cpuUso[0]
	dibujarBarra(t, 1, 4, ancho-2, "CPU", usoCPU,
		fmt.Sprintf("Uso: %.1f%%", usoCPU))

	// Info de red
	netInfo := fmt.Sprintf("Red - Recibido: %.2fMB  Enviado: %.2fMB",
		float64(netDatos[0].BytesRecv)/1e6,
		float64(netDatos[0].BytesSent)/1e6)
	dibujarTexto(t, 1, 7, netInfo, tcell.StyleDefault.Foreground(tcell.ColorGreen))

	// Interfaces de Red
	dibujarTexto(t, 1, 9, "Interfaces de Red:", tcell.StyleDefault.Foreground(tcell.ColorYellow))
	interfacesInfo := obtenerInfoInterfacesRed()
	for i, info := range interfacesInfo {
		if i > 3 {
			break // Limitar a mostrar solo las primeras 3 interfaces
		}
		dibujarTexto(t, 1, 10+i, info, tcell.StyleDefault.Foreground(tcell.ColorBlue))
	}

	// Procesos
	procesos := obtenerProcesos()
	dibujarTabla(t, 1, 15, ancho/2-1, alto-10, "+ RAM", ordenarPorRAM(procesos))
	dibujarTabla(t, ancho/2+1, 15, ancho/2-2, alto-10, "+ CPU", ordenarPorCPU(procesos))
	t.Show()
}

// Dibuja una barra visual de porcentaje
func dibujarBarra(t tcell.Screen, x, y, ancho int, titulo string, valor float64, texto string) {
	llenado := int(float64(ancho-2) * valor / 100)
	dibujarTexto(t, x, y, titulo+": "+texto, tcell.StyleDefault.Foreground(tcell.ColorYellow))
	t.SetContent(x, y+1, '[', nil, tcell.StyleDefault)
	for i := 0; i < ancho-2; i++ {
		simbolo := ' '
		estilo := tcell.StyleDefault.Background(tcell.ColorDarkGray)
		if i < llenado {
			estilo = tcell.StyleDefault.Background(tcell.ColorGreen)
		}
		t.SetContent(x+1+i, y+1, simbolo, nil, estilo)
	}
	t.SetContent(x+ancho-1, y+1, ']', nil, tcell.StyleDefault)
}

// Dibuja texto en pantalla
func dibujarTexto(t tcell.Screen, x, y int, texto string, estilo tcell.Style) {
	for i, letra := range texto {
		t.SetContent(x+i, y, letra, nil, estilo)
	}
}

// Muestra una tabla con procesos
func dibujarTabla(t tcell.Screen, x, y, ancho, alto int, titulo string, lista []Proceso) {
	dibujarTexto(t, x, y, titulo, tcell.StyleDefault.Foreground(tcell.ColorYellow))
	dibujarTexto(t, x, y+1, "PID   Nombre                CPU%     RAM%", tcell.StyleDefault.Foreground(tcell.ColorGreen))
	for i, p := range lista {
		if i >= alto-3 {
			break
		}
		// Cambio de color basado en el uso de CPU y RAM
		var estilo tcell.Style
		if p.CPU > 80 || p.RAM > 80 {
			estilo = tcell.StyleDefault.Foreground(tcell.ColorRed)
		} else if p.CPU > 50 || p.RAM > 50 {
			estilo = tcell.StyleDefault.Foreground(tcell.ColorYellow)
		} else {
			estilo = tcell.StyleDefault.Foreground(tcell.ColorGreen)
		}
		texto := fmt.Sprintf("%-6d %-20s %-8.1f %-8.1f", p.PID, cortar(p.Nombre, 20), p.CPU, p.RAM)
		dibujarTexto(t, x, y+2+i, texto, estilo)
	}
}

// Obtiene lista de procesos del sistema
func obtenerProcesos() []Proceso {
	lista, _ := process.Processes()
	var resultado []Proceso

	for _, p := range lista {
		nombre, _ := p.Name()
		cpuPorc, _ := p.CPUPercent()
		ramPorc, _ := p.MemoryPercent()
		ramInfo, _ := p.MemoryInfo()

		if ramInfo != nil {
			resultado = append(resultado, Proceso{
				PID:        p.Pid,
				Nombre:     nombre,
				CPU:        cpuPorc,
				RAM:        float64(ramPorc),
				TamanioRAM: ramInfo.RSS,
			})
		}
	}

	return resultado
}

// Ordena por RAM usada
func ordenarPorRAM(lista []Proceso) []Proceso {
	sort.Slice(lista, func(i, j int) bool {
		return lista[i].TamanioRAM > lista[j].TamanioRAM
	})
	if len(lista) > 10 {
		lista = lista[:10]
	}
	return lista
}

// Ordena por CPU usada
func ordenarPorCPU(lista []Proceso) []Proceso {
	sort.Slice(lista, func(i, j int) bool {
		return lista[i].CPU > lista[j].CPU
	})
	if len(lista) > 10 {
		lista = lista[:10]
	}
	return lista
}

// Recorta nombres largos
func cortar(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// Obtiene información detallada de las interfaces de red
func obtenerInfoInterfacesRed() []string {
	interfaces, _ := net.Interfaces()
	var info []string

	// Obtener estadísticas de tráfico de red
	netStats, _ := net.IOCounters(true)

	for _, iface := range interfaces {
		var ifaceInfo string
		// Mostrar solo interfaces activas
		if iface.Flags&net.FlagUp != 0 {
			// Buscar la interfaz en las estadísticas de tráfico
			for _, stat := range netStats {
				if stat.Name == iface.Name {
					ifaceInfo = fmt.Sprintf("%s - Recibido: %.2fMB, Enviado: %.2fMB, Paquetes: %d recibidos, %d enviados",
						iface.Name,
						float64(stat.BytesRecv)/1e6,
						float64(stat.BytesSent)/1e6,
						stat.PacketsRecv,
						stat.PacketsSent)

					// Agregar direcciones IP (si existen)
					addrs, _ := iface.Addrs()
					for _, addr := range addrs {
						ifaceInfo += fmt.Sprintf(", IP: %s", addr.String())
					}

					info = append(info, ifaceInfo)
					break
				}
			}
		}
	}
	return info
}

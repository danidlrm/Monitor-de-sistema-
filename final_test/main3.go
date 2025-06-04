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
	"github.com/shirou/gopsutil/v4/host"
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

	var frameCount int
	var lastSecond = time.Now()
	var fps int

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
				frameCount++
				now := time.Now()
				if now.Sub(lastSecond) >= time.Second {
					fps = frameCount
					frameCount = 0
					lastSecond = now
				}
				ancho, _ := t.Size()
				dibujarTexto(t, ancho-12, 0, fmt.Sprintf("FPS: %d", fps), tcell.StyleDefault.Foreground(tcell.Color104))
				t.Show()
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

func dibujarTitulo(t tcell.Screen, x, y int) {
	titulo := []string{

		"MONITOR DE SISTEMA",
		"",
		"by: Daniela Davila | Santiago Avila | Diego Vitela ",
		"",
		"Este proyecto es un monitor del sistema en tiempo real escrito en Go, disenado para ejecutarse en la terminal con una interfaz visual tipo consola usando la biblioteca tcell.",
		"Su proposito es ofrecer una vision clara y eficiente del rendimiento del sistema, mostrando el uso de CPU, memoria RAM, interfaces de red y los procesos más demandantes ordenados por consumo de recursos.",
		"",
		"",
		"-> Presiona ESC para salir.",
	}

	for i, linea := range titulo {
		dibujarTexto(t, x, y+i, linea, tcell.StyleDefault.Foreground(tcell.ColorWhite))
	}
}

// Actualiza todos los datos en pantalla
func actualizarPantalla(t tcell.Screen) {
	t.Clear()
	ancho, alto := t.Size()

	// Dibuja el título
	dibujarTitulo(t, 1, 40) // alineado a la izq.

	memY := 20
	dibujarTexto(t, 1, memY, "Información del sistema", tcell.StyleDefault.Foreground(tcell.ColorWhite))
	memoria, _ := mem.VirtualMemory()
	cpuUso, _ := cpu.Percent(0, false)
	netDatos, _ := net.IOCounters(false)

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
	maxInterfaces := 3
	if len(interfacesInfo) < maxInterfaces {
		maxInterfaces = len(interfacesInfo)
	}

	for i, info := range interfacesInfo {
		if i > 3 {
			break // Limitar a mostrar solo las primeras 3 interfaces
		}
		dibujarTexto(t, 1, 10+i, info, tcell.StyleDefault.Foreground(tcell.ColorBlue))
	}

	// Dibujar uptime justo después de interfaces
	lineaUptime := 11 + maxInterfaces
	dibujarTexto(t, 1, lineaUptime, obtenerUptime(), tcell.StyleDefault.Foreground(tcell.ColorGray))

	// Calcular la siguiente línea disponible para las tablas
	lineaInicioTablas := lineaUptime + 2
	altoTablas := alto - lineaInicioTablas - 2

	// Procesos
	procesos := obtenerProcesos()
	dibujarTabla(t, 1, 15, ancho/2-1, altoTablas, "+ RAM", ordenarPorRAM(procesos))
	dibujarTabla(t, ancho/2+1, 15, ancho/2-2, altoTablas, "+ CPU", ordenarPorCPU(procesos))
	t.Show()

}

// Dibuja una barra visual de porcentaje
func dibujarBarra(t tcell.Screen, x, y, ancho int, titulo string, valor float64, texto string) {
	llenado := int(float64(ancho-2) * valor / 100)
	dibujarTexto(t, x, y, titulo+": "+texto, tcell.StyleDefault.Foreground(tcell.ColorYellow))
	t.SetContent(x, y+1, '[', nil, tcell.StyleDefault)
	var color tcell.Color
	switch {
	case valor > 80:
		color = tcell.ColorRed
	case valor > 50:
		color = tcell.ColorYellow
	default:
		color = tcell.ColorGreen
	}
	for i := 0; i < ancho-2; i++ {
		simbolo := ' '
		estilo := tcell.StyleDefault.Background(tcell.ColorDarkGray)
		if i < llenado {
			estilo = tcell.StyleDefault.Background(color)
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
		// Mostrar solo interfaces activas (se omite net.FlagUp)
		for _, stat := range netStats {
			if stat.Name == iface.Name {

				ifaceInfo = fmt.Sprintf("%s - Recibido: %.2fMB, Enviado: %.2fMB, Paquetes: %d recibidos, %d enviados",

					iface.Name,
					float64(stat.BytesRecv)/1e6,
					float64(stat.BytesSent)/1e6,
					stat.PacketsRecv,
					stat.PacketsSent)

				// Agregar direcciones IP (si existen)
				for _, addr := range iface.Addrs {
					ifaceInfo += fmt.Sprintf(", IP: %s", addr.String())
				}

				info = append(info, ifaceInfo)
				break
			}
		}
	}
	return info
}

// timepo activo en sistema
func obtenerUptime() string {
	uptime, err := host.Uptime()
	if err != nil {
		return "Tiempo activo: No disponible"
	}
	duracion := time.Duration(uptime) * time.Second

	dias := int(duracion.Hours()) / 24
	horas := int(duracion.Hours()) % 24
	minutos := int(duracion.Minutes()) % 60
	segundos := int(duracion.Seconds()) % 60

	return fmt.Sprintf("Tiempo activo: %dd %02dh %02dm %02ds", dias, horas, minutos, segundos)
}

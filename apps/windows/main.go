// TECNO LAB para Windows
//
// Abre el laboratorio en su propia ventana (Microsoft Edge en modo aplicación; si no está,
// Google Chrome o el navegador predeterminado). Un servidor local, que solo atiende a este
// mismo equipo (127.0.0.1), entrega los archivos de la app, que van dentro del programa
// (comprimidos). Así funciona sin internet y sin instalar nada: la cámara, la realidad
// aumentada, los gestos con la mano (MediaPipe) y los modelos 3D funcionan igual que en la web.
//
// El programa se cierra solo cuando se cierra la ventana de TECNO LAB.
//
// Para generarlo (ver apps/README.md):
//
//	python3 src/construir.py --windows      (prepara apps/windows/www)
//	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H windowsgui" -o TECNO-LAB-Windows.exe
//
// Aplicación elaborada por: Jeniffer Andrea Fuenmayor Díaz · Docente asesor: José Zapata
package main

import (
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const version = "1.8.0"

//go:embed www
var archivos embed.FS

// Puertos fijos: así el navegador recuerda el progreso y el permiso de la cámara entre una vez y otra.
var puertos = []int{48731, 48732, 48733, 48734, 48735}

var tipos = map[string]string{
	".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8", ".css": "text/css; charset=utf-8",
	".json": "application/json; charset=utf-8", ".webmanifest": "application/manifest+json", ".wasm": "application/wasm",
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".webp": "image/webp", ".svg": "image/svg+xml",
	".ico": "image/x-icon", ".woff2": "font/woff2", ".pdf": "application/pdf", ".txt": "text/plain; charset=utf-8",
	".md": "text/plain; charset=utf-8", ".glb": "model/gltf-binary", ".data": "application/octet-stream",
	".tflite": "application/octet-stream", ".binarypb": "application/octet-stream",
}

// ---------- Ventanas abiertas: el programa termina cuando ya no queda ninguna ----------
type estado struct {
	mu       sync.Mutex
	ventanas map[string]time.Time
	ultimo   time.Time
	inicio   time.Time
	vista    bool
	salida   *time.Timer
}

var e = &estado{ventanas: map[string]time.Time{}, ultimo: time.Now(), inicio: time.Now()}

func (s *estado) actividad() {
	s.mu.Lock()
	s.ultimo = time.Now()
	if s.salida != nil {
		s.salida.Stop()
		s.salida = nil
	}
	s.mu.Unlock()
}

func (s *estado) vivo(id string) {
	s.mu.Lock()
	if id != "" {
		s.ventanas[id] = time.Now()
	}
	s.vista = true
	s.mu.Unlock()
}

// Al cerrar (o recargar) una ventana: si no queda ninguna y nadie vuelve a pedir nada en unos segundos, termina.
func (s *estado) cerrar(id string) {
	s.mu.Lock()
	delete(s.ventanas, id)
	if len(s.ventanas) == 0 {
		if s.salida != nil {
			s.salida.Stop()
		}
		s.salida = time.AfterFunc(6*time.Second, func() {
			s.mu.Lock()
			fin := len(s.ventanas) == 0 && time.Since(s.ultimo) >= 5*time.Second
			s.mu.Unlock()
			if fin {
				os.Exit(0)
			}
		})
	}
	s.mu.Unlock()
}

// Respaldo por si la ventana se cerró sin avisar (por ejemplo, si se apagó el navegador de golpe).
func (s *estado) vigilar() {
	for range time.Tick(10 * time.Second) {
		s.mu.Lock()
		for id, t := range s.ventanas {
			if time.Since(t) > 10*time.Minute {
				delete(s.ventanas, id)
			}
		}
		sinAbrir := !s.vista && time.Since(s.inicio) > 3*time.Minute
		abandonado := s.vista && len(s.ventanas) == 0 && time.Since(s.ultimo) > 2*time.Minute
		s.mu.Unlock()
		if sinAbrir || abandonado {
			os.Exit(0)
		}
	}
}

// ---------- Archivos ----------
func abrir(nombre string) (fs.File, int64, bool) {
	for _, gz := range []bool{true, false} {
		n := nombre
		if gz {
			n += ".gz"
		}
		if f, err := archivos.Open(n); err == nil {
			if st, err := f.Stat(); err == nil && !st.IsDir() {
				return f, st.Size(), gz
			}
			f.Close()
		}
	}
	return nil, 0, false
}

func servir(w http.ResponseWriter, r *http.Request) {
	e.actividad()
	p := path.Clean("/" + r.URL.Path)
	h := w.Header()
	switch p {
	case "/__tecnolab":
		h.Set("Cache-Control", "no-store")
		fmt.Fprint(w, "tecnolab "+version)
		return
	case "/__vivo":
		e.vivo(r.URL.Query().Get("id"))
		w.WriteHeader(http.StatusNoContent)
		return
	case "/__cerrar":
		e.cerrar(r.URL.Query().Get("id"))
		w.WriteHeader(http.StatusNoContent)
		return
	case "/__salir": // lo usa una versión más nueva del programa para reemplazar a esta
		w.WriteHeader(http.StatusNoContent)
		go func() { time.Sleep(300 * time.Millisecond); os.Exit(0) }()
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	if p == "/" {
		p = "/index.html"
	}
	f, tam, gz := abrir("www" + p)
	if f == nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	tipo := tipos[strings.ToLower(path.Ext(p))]
	if tipo == "" {
		tipo = "application/octet-stream"
	}
	h.Set("Content-Type", tipo)
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Content-Type-Options", "nosniff")
	etiqueta := `"` + version + "-" + strconv.FormatInt(tam, 36) + `"`
	h.Set("ETag", etiqueta)
	if r.Header.Get("If-None-Match") == etiqueta {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	var cuerpo io.Reader = f
	if gz {
		h.Set("Vary", "Accept-Encoding")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.Set("Content-Encoding", "gzip")
		} else { // navegador sin gzip (muy raro): se descomprime aquí
			datos, _ := io.ReadAll(f)
			zr, err := gzip.NewReader(bytes.NewReader(datos))
			if err != nil {
				http.Error(w, "Archivo dañado", http.StatusInternalServerError)
				return
			}
			plano, _ := io.ReadAll(zr)
			cuerpo, tam = bytes.NewReader(plano), int64(len(plano))
		}
	}
	h.Set("Content-Length", strconv.FormatInt(tam, 10))
	if r.Method == http.MethodHead {
		return
	}
	io.Copy(w, cuerpo)
}

// ---------- Ventana ----------
func navegadores() []string {
	var lista []string
	bases := []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")}
	for _, prog := range [][]string{{"Microsoft", "Edge", "Application", "msedge.exe"}, {"Google", "Chrome", "Application", "chrome.exe"}} {
		for _, b := range bases {
			if b != "" {
				lista = append(lista, filepath.Join(append([]string{b}, prog...)...))
			}
		}
	}
	return lista
}

func abrirVentana(dir string) {
	app := dir + "index.html?app=windows"
	if runtime.GOOS != "windows" {
		exec.Command("xdg-open", app).Start()
		return
	}
	for _, nav := range navegadores() {
		if _, err := os.Stat(nav); err == nil {
			if exec.Command(nav, "--app="+app, "--start-maximized", "--no-first-run", "--no-default-browser-check").Start() == nil {
				return
			}
		}
	}
	exec.Command("rundll32", "url.dll,FileProtocolHandler", app).Start() // navegador predeterminado
}

// ¿Hay un TECNO LAB abierto en ese puerto? Devuelve su versión ("" si no es TECNO LAB).
func consultar(p int, ruta string) string {
	c := http.Client{Timeout: 1500 * time.Millisecond}
	r, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d%s", p, ruta))
	if err != nil {
		return ""
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(r.Body, 64))
	return string(b)
}

func main() {
	sinVentana := len(os.Args) > 1 && os.Args[1] == "--sin-ventana" // para pruebas
	var ln net.Listener
	for _, p := range puertos {
		dir := fmt.Sprintf("127.0.0.1:%d", p)
		if l, err := net.Listen("tcp", dir); err == nil {
			ln = l
			break
		}
		switch v := consultar(p, "/__tecnolab"); {
		case v == "tecnolab "+version: // ya está abierto: solo se abre otra ventana
			if !sinVentana {
				abrirVentana("http://" + dir + "/")
			}
			return
		case strings.HasPrefix(v, "tecnolab "): // otra versión: se cierra y esta toma su lugar
			consultar(p, "/__salir")
			for i := 0; i < 20 && ln == nil; i++ {
				time.Sleep(150 * time.Millisecond)
				if l, err := net.Listen("tcp", dir); err == nil {
					ln = l
				}
			}
		}
		if ln != nil {
			break
		}
	}
	if ln == nil { // todos ocupados por otros programas: cualquier puerto libre
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			os.Exit(1)
		}
		ln = l
	}
	dir := "http://" + ln.Addr().String() + "/"
	go e.vigilar()
	if !sinVentana {
		go func() { time.Sleep(100 * time.Millisecond); abrirVentana(dir) }()
	} else {
		fmt.Println("TECNO LAB", version, "en", dir)
	}
	srv := &http.Server{Handler: http.HandlerFunc(servir), ReadHeaderTimeout: 15 * time.Second}
	srv.Serve(ln)
}

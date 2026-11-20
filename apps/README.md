# Apps de TECNO LAB

Las apps muestran exactamente el mismo laboratorio (`index.html` y sus carpetas), incluido dentro de la app para que funcione sin internet.

## Android (`android/`)

- `MainActivity.java`: abre el laboratorio en un WebView desde la carpeta interna `assets/www`.
- Pide el permiso de la cámara al abrirse y cada vez que la página lo necesita (realidad aumentada con cámara trasera o frontal, gestos con la mano y zona de creación). Si el permiso quedó bloqueado, abre los ajustes de la app para activarlo.
- Permite subir modelos .glb, guardar archivos en *Descargas › TECNO LAB* e imprimir el marcador.
- Paquete `edu.tecnolab.app`, versión 1.8.0, Android 7.0 o superior. El archivo listo para instalar está en `descargas/TECNO-LAB-Android.apk`.

## Windows (`windows/`)

- `main.go` (lenguaje [Go](https://go.dev), licencia BSD): un programa pequeño que lleva dentro el laboratorio (comprimido) y lo entrega con un servidor local que solo atiende a ese mismo computador (`127.0.0.1`). Abre TECNO LAB en su propia ventana con Microsoft Edge en modo aplicación (si no está, con Google Chrome o el navegador predeterminado).
- Funciona sin internet y sin instalar nada: la cámara, la realidad aumentada, los gestos con la mano y los modelos 3D funcionan igual que en la web. El progreso y el permiso de la cámara se recuerdan.
- Se cierra solo cuando se cierra la ventana de TECNO LAB. Si se abre dos veces, usa el que ya estaba abierto.
- Para generar `TECNO-LAB-Windows.exe` (con Go 1.22 o superior), desde la carpeta del proyecto:
  1. `python3 src/construir.py --windows` (copia y comprime la app en `apps/windows/www`).
  2. En `apps/windows`: `go mod init tecnolab` (solo la primera vez) y
     `GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H windowsgui" -o TECNO-LAB-Windows.exe`
  3. El icono y los datos del programa se agregan con [go-winres](https://github.com/tc-hib/go-winres): `go-winres simply --arch amd64 --icon ../../assets/icons/icon-512.png --manifest gui --product-name "TECNO LAB"` antes del paso 2.

## Escritorio con Electron (`escritorio/`)

- Versión alternativa hecha con [Electron](https://www.electronjs.org) (licencia MIT), útil para generar apps de macOS. Pesa más de 100 MB, por eso no va en la carpeta `descargas/`.
- `main.js` sirve los archivos del laboratorio desde la carpeta `www`, permite la cámara (en macOS la pide al abrirse) y abre los enlaces externos en el navegador.
- Para generar las apps: copiar el laboratorio en `escritorio/www/` y empaquetar con `@electron/packager`.

## Dónde se publican

La APK de Android y el programa de Windows pesan menos de 25 MB (el máximo que GitHub acepta por archivo desde el navegador), así que van en la carpeta `descargas/` del repositorio y la aplicación los ofrece en el menú **Más**. Archivos más grandes (por ejemplo, las apps de Electron) se suben a la sección **Releases** del repositorio, que acepta archivos de hasta 2 GB.

## iPhone y iPad

Apple solo permite instalar apps firmadas con una cuenta de Apple Developer desde un Mac. Mientras tanto, en Safari se abre la página publicada y se usa **Compartir → Agregar a pantalla de inicio**: queda como una app con su icono y funciona sin internet después de la primera visita.

---
Aplicación elaborada por: Jeniffer Andrea Fuenmayor Díaz · Docente asesor: José Zapata

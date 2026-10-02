//go:build desktopsmoke

// Native renderer smoke test. This program uses an isolated temporary database
// and is excluded from normal application builds and go test ./....
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/ui"
)

func main() {
	output := flag.String("output", "build/native-smoke", "Screenshot output directory")
	flag.Parse()
	root, err := os.MkdirTemp("", "notehub-native-smoke-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(root)
	backend, err := app.OpenBackend(context.Background(), app.Config{DataDir: root, Version: "smoke"})
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()
	if _, err := backend.Memos.Create(context.Background(), "Native desktop verification\nGhi chú tiếng Việt • FPGA #Research/FPGA"); err != nil {
		log.Fatal(err)
	}
	application := fyneapp.NewWithID("io.notehub.desktop.smoke")
	desktop := ui.NewWindow(application, backend, "smoke")
	desktop.SetTitle("NoteHub · native smoke test · temporary data")
	// Keep the QA window inside a 1080p display even at Windows 130% scaling.
	// The production window still opens at its specified 1450 × 900 size.
	desktop.Resize(fyne.NewSize(1280, 730))
	desktop.CenterOnScreen()
	outcome := make(chan error, 1)
	application.Lifecycle().SetOnStarted(func() {
		go func() {
			result := func() error {
				if err := os.MkdirAll(*output, 0o755); err != nil {
					return err
				}
				for _, page := range []string{"home", "calendar", "search", "attachments", "tags", "settings"} {
					fyne.DoAndWait(func() { desktop.Navigate(page) })
					deadline := time.Now().Add(15 * time.Second)
					for desktop.Jobs.Busy() {
						if time.Now().After(deadline) {
							return fmt.Errorf("%s did not finish loading", page)
						}
						time.Sleep(20 * time.Millisecond)
					}
					// Flush callbacks posted by workers before asserting/rendering.
					fyne.DoAndWait(func() {})
				}
				for _, mode := range []string{"Light", "Dark"} {
					fyne.DoAndWait(func() {
						desktop.Navigate("home")
						application.Settings().SetTheme(ui.NewTheme(mode))
					})
					if err := captureSettled(desktop, filepath.Join(*output, "native-"+mode+".png")); err != nil {
						return fmt.Errorf("%s: %w", mode, err)
					}
				}
				return nil
			}()
			outcome <- result
			fyne.Do(func() { desktop.Window.SetCloseIntercept(nil); desktop.Window.Close() })
		}()
	})
	desktop.ShowAndRun()
	if err := desktop.Wait(); err != nil {
		log.Fatal(err)
	}
	if err := <-outcome; err != nil {
		log.Fatal(err)
	}
	fmt.Println("NATIVE_DESKTOP_SMOKE_OK: six screens, Light/Dark rendering, orderly shutdown")
}

// Capture reads the last OpenGL frame. A flushed event queue alone does not
// guarantee that layout/theme changes have reached that frame yet.
func captureSettled(desktop *ui.Desktop, path string) error {
	deadline := time.Now().Add(15 * time.Second)
	var previous [32]byte
	stable := 0
	for time.Now().Before(deadline) {
		time.Sleep(150 * time.Millisecond)
		if desktop.Jobs.Busy() {
			stable = 0
			continue
		}
		var encoded bytes.Buffer
		var captureErr error
		fyne.DoAndWait(func() {
			captured := desktop.Canvas().Capture()
			if captured == nil || captured.Bounds().Dx() < 1000 {
				captureErr = fmt.Errorf("native canvas was not rendered")
				return
			}
			// The desktop framebuffer is already composited. Its alpha channel
			// is not PNG transparency (notably around antialiased rectangles).
			opaque := image.NewRGBA(captured.Bounds())
			for y := opaque.Rect.Min.Y; y < opaque.Rect.Max.Y; y++ {
				for x := opaque.Rect.Min.X; x < opaque.Rect.Max.X; x++ {
					r, g, b, _ := captured.At(x, y).RGBA()
					opaque.SetRGBA(x, y, color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255})
				}
			}
			captureErr = png.Encode(&encoded, opaque)
		})
		if captureErr != nil {
			return captureErr
		}
		current := sha256.Sum256(encoded.Bytes())
		if current == previous {
			stable++
		} else {
			stable = 0
		}
		previous = current
		if stable >= 2 && !desktop.Jobs.Busy() {
			return os.WriteFile(path, encoded.Bytes(), 0o600)
		}
	}
	return fmt.Errorf("native render did not settle before timeout")
}

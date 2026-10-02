// Package assets embeds resources so launching from any directory works.
package assets

import (
 _ "embed"
 "fyne.io/fyne/v2"
)

//go:embed icons/NoteHub.png
var logo []byte

var Logo = fyne.NewStaticResource("NoteHub.png",logo)

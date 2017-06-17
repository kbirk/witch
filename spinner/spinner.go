package spinner

import (
	"fmt"
	"io"
	"math/rand"

	"github.com/fatih/color"

	"github.com/unchartedsoftware/witch/cursor"
)

var (
	wand = fmt.Sprintf("%s%s", color.GreenString("--"), color.MagentaString("⭑"))
	// `°º¤ø,¸¸,ø¤º°`
	frames = []string{
		"",
		"°",
		"°º",
		"°º¤",
		" º¤ø",
		"  ¤ø,",
		"   ø,¸",
		"    ,¸¸",
		"     ¸¸,",
		"      ¸,ø",
		"       ,ø¤",
		"        ø¤º",
		"         ¤º°",
		"          º°`",
		"           °``",
		"            ``°",
		"             `°º",
		"              °º¤",
		"               º¤ø",
		"                ¤ø,",
		"                 ø,¸",
		"                  ,¸¸",
		"                   ¸¸,",
		"                    ¸,",
		"                     ,",
	}
)

// Spinner represents a spinning console output.
type Spinner struct {
	c int
	w io.Writer
}

// New instantiates and returns a new spinner struct.
func New(writer io.Writer) *Spinner {
	return &Spinner{
		w: writer,
	}
}

// Tick increments the cursor.
func (s *Spinner) Tick(count uint64) {
	s.c = (s.c + 1) % len(frames)
	s.w.Write([]byte(cursor.ClearLine()))
	magic := fmt.Sprintf("%s%s %s %s",
		cursor.HideCursor(),
		fileCountString(count),
		wand,
		randomColors(frames[s.c]))
	s.w.Write([]byte(magic))
}

// Done clears the cursor.
func (s *Spinner) Done() {
	s.w.Write([]byte(fmt.Sprintf("watch terminated %s%s",
		color.GreenString("✘"),
		cursor.ShowCursor())))
}

func randomColors(str string) string {
	res := ""
	for _, c := range str {
		s := string(c)
		if s == " " {
			res += " "
		} else {
			i := int(rand.Float64() * 4)
			switch i {
			case 0:
				res += color.MagentaString(s)
			case 1:
				res += color.BlueString(s)
			case 2:
				res += color.GreenString(s)
			case 3:
				res += color.CyanString(s)
			}
		}
	}
	return res
}

func fileCountString(count uint64) string {
	switch count {
	case 0:
		return color.BlackString("no files found")
	case 1:
		return fmt.Sprintf("%s %s %s",
			color.BlackString("watching"),
			color.CyanString("%d", count),
			color.BlackString("file"))
	}
	return fmt.Sprintf("%s %s %s",
		color.BlackString("watching"),
		color.CyanString("%d", count),
		color.BlackString("files"))
}

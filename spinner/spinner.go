package spinner

import (
	"fmt"
	"io"
	"math/rand"

	"github.com/fatih/color"

	"github.com/unchartedsoftware/witch/cursor"
)

var (
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
	s.w.Write([]byte(cursor.Hide()))
	s.w.Write([]byte(castMagic(frames[s.c])))

}

// Done clears the cursor.
func (s *Spinner) Done() {
	s.w.Write([]byte(fmt.Sprintf("watch terminated %s%s",
		color.GreenString("✘"),
		cursor.Show())))
}

func castMagic(str string) string {
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

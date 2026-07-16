package output

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/cli/internal/api"
	"github.com/EOEboh/hookdrop/cli/internal/forward"
)

// Printer serializes all terminal writes through one goroutine so event
// lines, forward results, and status messages never interleave, no matter
// how fast webhooks arrive.
type Printer struct {
	lines  chan string
	done   chan struct{}
	Colors bool
}

func NewPrinter() *Printer {
	p := &Printer{
		lines:  make(chan string, 256),
		done:   make(chan struct{}),
		Colors: ColorsEnabled(),
	}
	go func() {
		defer close(p.done)
		for line := range p.lines {
			fmt.Fprintln(os.Stdout, line)
		}
	}()
	return p
}

// Line queues a line for printing.
func (p *Printer) Line(s string) {
	p.lines <- s
}

// Close flushes queued lines and stops the writer.
func (p *Printer) Close() {
	close(p.lines)
	<-p.done
}

// Event renders the one-line summary of a captured webhook:
//
//	12:04:31  POST    ✓ stripe  1.2 KB
//
// showID appends a short request id used to correlate forward results.
func (p *Printer) Event(req *api.CapturedRequest, showID bool) string {
	ts := Colorize(p.Colors, Dim, req.ReceivedAt.Local().Format("15:04:05"))
	method := Colorize(p.Colors, MethodStyle(req.Method), fmt.Sprintf("%-6s", req.Method))

	var verify string
	switch req.Verified {
	case "verified":
		verify = Colorize(p.Colors, Green, "✓")
	case "failed":
		verify = Colorize(p.Colors, Red, "✗")
	default:
		verify = Colorize(p.Colors, Dim, "–")
	}
	if req.Provider != "" {
		verify += " " + Colorize(p.Colors, Dim, req.Provider)
	}

	line := fmt.Sprintf("%s  %s  %s  %s", ts, method, verify, FormatSize(req.BodySize))
	if showID {
		line += "  " + Colorize(p.Colors, Dim, "("+ShortID(req.ID)+")")
	}
	return line
}

// ForwardResult renders the delivery outcome, indented under its event:
//
//	   ↳ (4dbb48bd) 200 in 45ms
func (p *Printer) ForwardResult(res forward.Result) string {
	id := Colorize(p.Colors, Dim, "("+ShortID(res.Request.ID)+")")
	if res.Err != nil {
		return fmt.Sprintf("   ↳ %s %s", id, Colorize(p.Colors, Red, "forward failed: "+res.Err.Error()))
	}

	style := Green
	if res.Status >= 500 {
		style = Red
	} else if res.Status >= 400 {
		style = Yellow
	}
	return fmt.Sprintf("   ↳ %s %s in %s",
		id,
		Colorize(p.Colors, style, fmt.Sprintf("%d", res.Status)),
		res.Latency.Round(time.Millisecond),
	)
}

// Status renders dim informational lines (connecting, reconnecting…).
func (p *Printer) Status(s string) string {
	return Colorize(p.Colors, Dim, s)
}

// Ready renders the multi-line banner shown once the stream connects.
func (p *Printer) Ready(inboxURL, forwardURL string) string {
	check := Colorize(p.Colors, Green, "✓")
	var b strings.Builder
	fmt.Fprintf(&b, "%s Ready — listening on %s\n", check, Colorize(p.Colors, Bold, inboxURL))
	if forwardURL != "" {
		fmt.Fprintf(&b, "  → forwarding to %s\n", Colorize(p.Colors, Bold, forwardURL))
	}
	b.WriteString(Colorize(p.Colors, Dim, "  waiting for webhooks…  (Ctrl-C to stop)"))
	return b.String()
}

func ShortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func FormatSize(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

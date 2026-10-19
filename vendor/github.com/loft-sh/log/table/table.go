package table

import (
	"io"

	"github.com/fatih/color"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/sirupsen/logrus"
)

func PrintTable(s log.Logger, header []string, values [][]string) {
	PrintTableWithOptions(s, header, values, nil)
}

// PrintTableWithOptions prints a table with header columns and string values
func PrintTableWithOptions(
	s log.Logger,
	header []string,
	values [][]string,
	modify func(table *tablewriter.Table),
) {
	reader, writer := io.Pipe()
	defer writer.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)

		sa := scanner.NewScanner(reader)
		for sa.Scan() {
			s.WriteString(logrus.InfoLevel, "  "+sa.Text()+"\n")
		}
	}()

	table := tablewriter.NewTable(writer,
		tablewriter.WithRenderer(colorHeaderRenderer{Renderer: renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Symbols: tw.NewSymbols(tw.StyleASCII),
		})}),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Header(header)
	if err := table.Bulk(values); err != nil {
		s.Debugf("append table rows: %v", err)
	}
	if modify != nil {
		modify(table)
	}

	// Render
	_, _ = writer.Write([]byte("\n"))
	if err := table.Render(); err != nil {
		s.Debugf("render table: %v", err)
	}
	_, _ = writer.Write([]byte("\n"))
	_ = writer.Close()
	<-done
}

type colorHeaderRenderer struct {
	tw.Renderer
}

func (r colorHeaderRenderer) Header(header [][]string, context tw.Formatting) {
	green := color.New(color.FgGreen)
	coloredHeader := make([][]string, len(header))
	for row, values := range header {
		coloredHeader[row] = make([]string, len(values))
		for column, value := range values {
			coloredHeader[row][column] = green.Sprint(value)
		}
	}

	coloredContext := context
	coloredContext.Row.Current = make(map[int]tw.CellContext, len(context.Row.Current))
	for column, cell := range context.Row.Current {
		cell.Data = green.Sprint(cell.Data)
		coloredContext.Row.Current[column] = cell
	}

	r.Renderer.Header(coloredHeader, coloredContext)
}

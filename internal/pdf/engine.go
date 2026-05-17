package pdf

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/url"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type EngineDeps struct {
	ChromeExecutablePath string
	Timeout              time.Duration
}

type Engine struct {
	chromePath string
	timeout    time.Duration
}

func NewEngine(deps EngineDeps) *Engine {
	return &Engine{chromePath: deps.ChromeExecutablePath, timeout: deps.Timeout}
}

type RenderInput struct {
	HTML string
}

func (e *Engine) RenderHTMLToPDF(ctx context.Context, in RenderInput) ([]byte, error) {
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	if e.chromePath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(e.chromePath))
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, allocOpts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	var pdfBuf []byte
	err := chromedp.Run(browserCtx,
		chromedp.Navigate("data:text/html;charset=utf-8,"+url.PathEscape(in.HTML)),
		chromedp.Sleep(250*time.Millisecond),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				WithMarginTop(0).
				WithMarginBottom(0).
				WithMarginLeft(0).
				WithMarginRight(0).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("render pdf: %w", err)
	}
	return pdfBuf, nil
}

func MustParseTemplate(name, content string) *template.Template {
	t, err := template.New(name).Parse(content)
	if err != nil {
		panic(err)
	}
	return t
}

func RenderTemplate(t *template.Template, data any) (string, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

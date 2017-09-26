// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package render

import (
	"bytes"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/CloudVLab/tools/claat/types"
)

// TODO: render Qwiklabs HTML using golang/x/net/html or template.

// QwiklabsHTML renders nodes as the markup for the target env.
func QwiklabsHTML(env string, nodes ...types.Node) (htmlTemplate.HTML, error) {
	var buf bytes.Buffer
	if err := WriteQwiklabsHTML(&buf, env, nodes...); err != nil {
		return "", err
	}
	return htmlTemplate.HTML(buf.String()), nil
}

// WriteQwiklabsHTML does the same as Qwiklabs but outputs rendered markup to w.
func WriteQwiklabsHTML(w io.Writer, env string, nodes ...types.Node) error {
	qw := qwiklabsHTMLWriter{w: w, env: env}
	return qw.write(nodes...)
}

type qwiklabsHTMLWriter struct {
	w   io.Writer // output writer
	env string    // target environment
	err error     // error during any writeXxx methods
}

func (qw *qwiklabsHTMLWriter) matchEnv(v []string) bool {
	if len(v) == 0 || qw.env == "" {
		return true
	}
	i := sort.SearchStrings(v, qw.env)
	return i < len(v) && v[i] == qw.env
}

func (qw *qwiklabsHTMLWriter) write(nodes ...types.Node) error {
	for _, n := range nodes {
		if !qw.matchEnv(n.Env()) {
			continue
		}
		switch n := n.(type) {
		case *types.TextNode:
			qw.text(n)
		case *types.ImageNode:
			qw.image(n)
		case *types.URLNode:
			qw.url(n)
		case *types.ButtonNode:
			qw.button(n)
		case *types.CodeNode:
			qw.code(n)
			qw.writeBytes(newLine)
		case *types.ListNode:
			qw.list(n)
			qw.writeBytes(newLine)
		case *types.ImportNode:
			if len(n.Content.Nodes) == 0 {
				break
			}
			qw.list(n.Content)
			qw.writeBytes(newLine)
		case *types.ItemsListNode:
			qw.itemsList(n)
			qw.writeBytes(newLine)
		case *types.GridNode:
			qw.grid(n)
			qw.writeBytes(newLine)
		case *types.InfoboxNode:
			qw.infobox(n)
			qw.writeBytes(newLine)
		case *types.SurveyNode:
			qw.survey(n)
			qw.writeBytes(newLine)
		case *types.HeaderNode:
			qw.header(n)
			qw.writeBytes(newLine)
		case *types.YouTubeNode:
			qw.youtube(n)
			qw.writeBytes(newLine)
		}
		if qw.err != nil {
			return qw.err
		}
	}
	return nil
}

func (qw *qwiklabsHTMLWriter) writeBytes(b []byte) {
	if qw.err != nil {
		return
	}
	_, qw.err = qw.w.Write(b)
}

func (qw *qwiklabsHTMLWriter) writeString(s string) {
	qw.writeBytes([]byte(s))
}

func (qw *qwiklabsHTMLWriter) writeFmt(f string, a ...interface{}) {
	qw.writeString(fmt.Sprintf(f, a...))
}

func (qw *qwiklabsHTMLWriter) writeEscape(s string) {
	htmlTemplate.HTMLEscape(qw.w, []byte(s))
}

func (qw *qwiklabsHTMLWriter) text(n *types.TextNode) {
	if n.Bold {
		qw.writeString("<strong>")
	}
	if n.Italic {
		qw.writeString("<em>")
	}
	if n.Code {
		qw.writeString("<code>")
	}
	s := htmlTemplate.HTMLEscapeString(n.Value)
	qw.writeString(strings.Replace(s, "\n", "<br>", -1))
	if n.Code {
		qw.writeString("</code>")
	}
	if n.Italic {
		qw.writeString("</em>")
	}
	if n.Bold {
		qw.writeString("</strong>")
	}
}

func (qw *qwiklabsHTMLWriter) image(n *types.ImageNode) {
	qw.writeString("<img")
	if n.MaxWidth > 0 {
		qw.writeFmt(` style="max-width: %.2fpx"`, n.MaxWidth)
	}
	qw.writeString(` src="`)
	qw.writeString(n.Src)
	qw.writeBytes(doubleQuote)
	qw.writeBytes(greaterThan)
}

func (qw *qwiklabsHTMLWriter) url(n *types.URLNode) {
	qw.writeString("<a")
	if n.URL != "" {
		qw.writeString(` href="`)
		qw.writeString(n.URL)
		qw.writeBytes(doubleQuote)
	}
	if n.Name != "" {
		qw.writeString(` name="`)
		qw.writeEscape(n.Name)
		qw.writeBytes(doubleQuote)
	}
	if n.Target != "" {
		qw.writeString(` target="`)
		qw.writeEscape(n.Target)
		qw.writeBytes(doubleQuote)
	}
	qw.writeBytes(greaterThan)
	qw.write(n.Content.Nodes...)
	qw.writeString("</a>")
}

func (qw *qwiklabsHTMLWriter) button(n *types.ButtonNode) {
	qw.writeString("<button")
	if n.Colored {
		qw.writeString(` class="codelabs-downloadbutton"`)
	}
	if n.Raised {
		qw.writeString(" raised")
	}
	qw.writeBytes(greaterThan)
	if n.Download {
		qw.writeString(`<i class="material-icons">file_download</i>`)
	}
	qw.write(n.Content.Nodes...)
	qw.writeString("</button>")
}

func (qw *qwiklabsHTMLWriter) code(n *types.CodeNode) {
	qw.writeString(`<pre class="prettyprint">`)
	if !n.Term {
		qw.writeString("<code")
		if n.Lang != "" {
			qw.writeFmt(" language=%q class=%q", n.Lang, n.Lang)
		}
		qw.writeBytes(greaterThan)
	}
	qw.writeEscape(n.Value)
	if !n.Term {
		qw.writeString("</code>")
	}
	qw.writeString("</pre>")
}

func (qw *qwiklabsHTMLWriter) list(n *types.ListNode) {
	wrap := n.Block() == true
	if wrap {
		qw.writeString("<p>")
	}
	qw.write(n.Nodes...)
	if wrap {
		qw.writeString("</p>")
	}
}

func (qw *qwiklabsHTMLWriter) itemsList(n *types.ItemsListNode) {
	tag := "ul"
	if n.Type() == types.NodeItemsList && n.Start > 0 {
		tag = "ol"
	}
	qw.writeBytes(lessThan)
	qw.writeString(tag)
	switch n.Type() {
	case types.NodeItemsCheck:
		qw.writeString(` class="checklist"`)
	case types.NodeItemsFAQ:
		qw.writeString(` class="faq"`)
	default:
		if n.ListType != "" {
			qw.writeString(` type="`)
			qw.writeString(n.ListType)
			qw.writeBytes(doubleQuote)
		}
		if n.Start > 0 {
			qw.writeFmt(` start="%d"`, n.Start)
		}
	}
	qw.writeBytes(greaterThan)
	qw.writeBytes(newLine)

	for _, i := range n.Items {
		qw.writeString("<li>")
		qw.write(i.Nodes...)
		qw.writeString("</li>\n")
	}

	qw.writeString("</")
	qw.writeString(tag)
	qw.writeBytes(greaterThan)
}

func (qw *qwiklabsHTMLWriter) grid(n *types.GridNode) {
	qw.writeString("<table>\n")
	for _, r := range n.Rows {
		qw.writeString("<tr>")
		for _, c := range r {
			qw.writeFmt(`<td colspan="%d" rowspan="%d">`, c.Colspan, c.Rowspan)
			qw.write(c.Content.Nodes...)
			qw.writeString("</td>")
		}
		qw.writeString("</tr>\n")
	}
	qw.writeString("</table>")
}

func (qw *qwiklabsHTMLWriter) infobox(n *types.InfoboxNode) {
	qw.writeString(`<aside class="`)
	qw.writeEscape(string(n.Kind))
	qw.writeString(`">`)
	qw.write(n.Content.Nodes...)
	qw.writeString("</aside>")
}

func (qw *qwiklabsHTMLWriter) survey(n *types.SurveyNode) {
	// We don't support surveys right now. Checkout `html.go` when we feel like
	// adding them back.
}

func (qw *qwiklabsHTMLWriter) header(n *types.HeaderNode) {
	// GDocs have "Title" and then "Heading {1|2|3}". We want to convert this to
	// HTML has "Title" => "h1", "Heading 1" => "h2", and so on. Note that
	// "Title" and "Heading 1" actually denote lab titles and step titles and are
	// handled in `template-qwiklabs.html` and not this function.
	tag := "h" + strconv.Itoa(n.Level+1)
	qw.writeBytes(lessThan)
	qw.writeString(tag)
	switch n.Type() {
	case types.NodeHeaderCheck:
		qw.writeString(` class="checklist"`)
	case types.NodeHeaderFAQ:
		qw.writeString(` class="faq"`)
	}
	qw.writeBytes(greaterThan)
	qw.write(n.Content.Nodes...)
	qw.writeString("</")
	qw.writeString(tag)
	qw.writeBytes(greaterThan)
}

func (qw *qwiklabsHTMLWriter) youtube(n *types.YouTubeNode) {
	// We don't support YT videos right now. Checkout `html.go` when we feel like
	// adding them back.
}

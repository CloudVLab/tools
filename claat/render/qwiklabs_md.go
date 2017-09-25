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
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	htmlTemplate "html/template"

	"github.com/CloudVLab/tools/claat/types"
)

// QwiklabsMD renders nodes as markdown for the target env.
func QwiklabsMD(env string, nodes ...types.Node) (string, error) {
	var buf bytes.Buffer
	if err := WriteQwiklabsMD(&buf, env, nodes...); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteQwiklabsMD does the same as MD but outputs rendered markup to w.
func WriteQwiklabsMD(w io.Writer, env string, nodes ...types.Node) error {
	qw := qwiklabsMDWriter{w: w, env: env}
	return qw.write(nodes...)
}

type qwiklabsMDWriter struct {
	w         io.Writer // output writer
	env       string    // target environment
	err       error     // error during any writeXxx methods
	lineStart bool
}

func (qw *qwiklabsMDWriter) writeBytes(b []byte) {
	if qw.err != nil {
		return
	}
	qw.lineStart = len(b) > 0 && b[len(b)-1] == '\n'
	_, qw.err = qw.w.Write(b)
}

func (qw *qwiklabsMDWriter) writeString(s string) {
	qw.writeBytes([]byte(s))
}

func (qw *qwiklabsMDWriter) writeFmt(f string, a ...interface{}) {
	qw.writeString(fmt.Sprintf(f, a...))
}

func (qw *qwiklabsMDWriter) writeEscape(s string) {
	htmlTemplate.HTMLEscape(qw.w, []byte(s))
}

func (qw *qwiklabsMDWriter) space() {
	if !qw.lineStart {
		qw.writeString(" ")
	}
}

func (qw *qwiklabsMDWriter) newBlock() {
	if !qw.lineStart {
		qw.writeBytes(newLine)
	}
	qw.writeBytes(newLine)
}

func (qw *qwiklabsMDWriter) matchEnv(v []string) bool {
	if len(v) == 0 || qw.env == "" {
		return true
	}
	i := sort.SearchStrings(v, qw.env)
	return i < len(v) && v[i] == qw.env
}

func (qw *qwiklabsMDWriter) write(nodes ...types.Node) error {
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
			// This can only be called directly when the button has no URL. The parser
			// wraps all buttons with URLs in a URLNode, which `qw.url(...)` will
			// handle directly.
			// WARNING: Currently, a button without a URL is not well specified behavior.
			qw.button(n, "")
		case *types.CodeNode:
			qw.code(n)
		case *types.ListNode:
			qw.list(n)
		case *types.ImportNode:
			if len(n.Content.Nodes) == 0 {
				break
			}
			qw.write(n.Content.Nodes...)
		case *types.ItemsListNode:
			qw.itemsList(n)
		case *types.GridNode:
			qw.grid(n)
		case *types.InfoboxNode:
			qw.infobox(n)
		//case *types.SurveyNode:
		//	qw.survey(n)
		case *types.HeaderNode:
			qw.header(n)
			//case *types.YouTubeNode:
			//	qw.youtube(n)
		}
		if qw.err != nil {
			return qw.err
		}
	}
	return nil
}

func (qw *qwiklabsMDWriter) text(n *types.TextNode) {
	if n.Bold {
		qw.writeString("__")
	}
	if n.Italic {
		qw.writeString("*")
	}
	if n.Code {
		qw.writeString("`")
	}
	qw.writeString(n.Value)
	if n.Code {
		qw.writeString("`")
	}
	if n.Italic {
		qw.writeString("*")
	}
	if n.Bold {
		qw.writeString("__")
	}
}

func (qw *qwiklabsMDWriter) image(n *types.ImageNode) {
	qw.space()
	qw.writeString("![")
	qw.writeString(path.Base(n.Src))
	qw.writeString("](")
	qw.writeString(n.Src)
	qw.writeString(")")
}

func (qw *qwiklabsMDWriter) url(n *types.URLNode) {
	// TODO: This code is bending over backwards to handle URLNodes that
	//  contain a ButtonNode. The gdoc parser will do this for all ButtonNodes
	//  it finds in a document, which makes sense for the HTML renderer, but
	//  adds complexity to any Markdown rendering format... which is probably
	//  the right trade-off choice since Markdown has no concept of buttons.
	for _, cn := range n.Content.Nodes {
		if childButton, ok := cn.(*types.ButtonNode); ok {
			qw.button(childButton, n.URL)
			return
		}
	}

	qw.space()
	if n.URL != "" {
		qw.writeString("[")
	}
	for _, cn := range n.Content.Nodes {
		if t, ok := cn.(*types.TextNode); ok {
			qw.writeString(t.Value)
		}
	}
	if n.URL != "" {
		qw.writeString("](")
		qw.writeString(n.URL)
		qw.writeString(")")
	}
}

func (qw *qwiklabsMDWriter) button(n *types.ButtonNode, url string) {
	if url == "" {
		url = "#"
	}

	qw.space()
	qw.writeFmt("<a class=\"codelabs-downloadbutton\" href=\"%s\" target=\"_blank\">", url)
	for _, cn := range n.Content.Nodes {
		if t, ok := cn.(*types.TextNode); ok {
			qw.writeString(t.Value)
		}
	}
	qw.writeString("</a>")
}

func (qw *qwiklabsMDWriter) code(n *types.CodeNode) {
	qw.newBlock()
	qw.writeBytes(newLine)
	defer qw.writeBytes(newLine)

	lang := n.Lang
	if n.Term {
		lang = "bash"
	}

	// TODO: There used to be a distinction between terminal commands and
	//   other code blocks. Terminal used the indented format.

	qw.writeString("```")
	qw.writeString(lang)
	qw.writeBytes(newLine)
	qw.writeString(n.Value)
	if !qw.lineStart {
		qw.writeBytes(newLine)
	}
	qw.writeString("```")
}

func (qw *qwiklabsMDWriter) list(n *types.ListNode) {
	if n.Block() == true {
		qw.newBlock()
	}
	qw.write(n.Nodes...)
	if !qw.lineStart {
		qw.writeBytes(newLine)
	}
}

func (qw *qwiklabsMDWriter) itemsList(n *types.ItemsListNode) {
	qw.newBlock()
	for i, item := range n.Items {
		s := "* "
		if n.Type() == types.NodeItemsList && n.Start > 0 {
			s = strconv.Itoa(i+n.Start) + ". "
		}
		qw.writeString(s)
		qw.write(item.Nodes...)
		if !qw.lineStart {
			qw.writeBytes(newLine)
		}
	}
}

func (qw *qwiklabsMDWriter) grid(n *types.GridNode) {
	// Note: There is no defined mapping of a google doc table to any default
	//   Markdown syntax. We have decided to mix raw HTML into our Qwiklabs
	//   Markdown documents.
	// TODO: Extend the Markdown syntax more rigorously.
	qw.newBlock()
	qw.writeString("<table>\n")
	for _, r := range n.Rows {
		qw.writeString("<tr>")
		for _, c := range r {
			qw.writeFmt(`<td colspan="%d" rowspan="%d">`, c.Colspan, c.Rowspan)
			// Use the existing HTML writer to transform the infobox body content.
			WriteHTML(qw.w, qw.env, c.Content.Nodes...)
			qw.writeString("</td>")
		}
		qw.writeString("</tr>\n")
	}
	qw.writeString("</table>")
}

func (qw *qwiklabsMDWriter) infobox(n *types.InfoboxNode) {
	// Note: There is no defined mapping of a Codelabs info box to any default
	//   Markdown syntax. We have decided to mix raw HTML into our Qwiklabs
	//   Markdown documents.
	// TODO: Extend the Markdown syntax more rigorously.
	qw.newBlock()
	qw.writeString(`<div class="codelabs-infobox codelabs-infobox-`)
	qw.writeEscape(string(n.Kind))
	qw.writeString(`">`)

	// Use the existing HTML writer to transform the infobox body content.
	WriteHTML(qw.w, qw.env, n.Content.Nodes...)

	qw.writeString("</div>")
}

func (qw *qwiklabsMDWriter) header(n *types.HeaderNode) {
	qw.newBlock()
	// This used to be `n.Level+1` so H1 => "##", this makes sense because the
	// lab's title is rendered as a "#", so everything else shifts down one.
	// That scheme makes more sense than having H1s map to the same level as the
	// title, but we need this match the style guide from existing AWS labs... ugh
	qw.writeString(strings.Repeat("#", n.Level))
	qw.writeString(" ")
	qw.write(n.Content.Nodes...)
	if !qw.lineStart {
		qw.writeBytes(newLine)
	}
}

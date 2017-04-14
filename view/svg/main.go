// Copyright 2017 Google Inc.
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

package main

import (
	"github.com/gopherjs/gopherjs/js"

	"github.com/google/shenzhen-go/api"
)

const (
	activeColour = "#09f"
	normalColour = "#000"
	errorColour  = "#f06"

	errRectStyle = "fill: #fee; fill-opacity: 0.5; stroke: #533; stroke-width:1"
	errTextStyle = "font-family:Go; font-size:16; user-select:none; pointer-events:none"

	pinRadius = 5
	lineWidth = 2
	snapQuad  = 144
)

var (
	document   = js.Global.Get("document")
	diagramSVG = document.Call("getElementById", "diagram")
	svgNS      = diagramSVG.Get("namespaceURI")
	graphPath  = js.Global.Get("graphPath").String()
	apiURL     = js.Global.Get("apiURL").String()

	saveGraphPropsLink = document.Call("getElementById", "save-graph-properties")

	errLabel *textBox

	dragItem draggable

	graph *Graph

	client api.Interface
)

func makeSVGElement(n string) *js.Object { return document.Call("createElementNS", svgNS, n) }
func cursorPos(e *js.Object) (x, y float64) {
	bcr := diagramSVG.Call("getBoundingClientRect")
	x = e.Get("clientX").Float() - bcr.Get("left").Float()
	y = e.Get("clientY").Float() - bcr.Get("top").Float()
	return
}

// Point is anything that has a position on the canvas.
type Point interface {
	Pt() (x, y float64)
}

type ephemeral struct{ x, y float64 }

func (e ephemeral) Pt() (x, y float64) { return e.x, e.y }

type draggable interface {
	drag(*js.Object)
	drop(*js.Object)
}

func mouseMove(e *js.Object) {
	if dragItem == nil {
		return
	}
	dragItem.drag(e)
}

func mouseUp(e *js.Object) {
	if dragItem == nil {
		return
	}
	dragItem.drag(e)
	dragItem.drop(e)
	dragItem = nil
}

func setError(err string, x, y float64) {
	if err == "" {
		clearError()
		return
	}
	diagramSVG.Call("appendChild", errLabel.group) // Bring to front
	errLabel.moveTo(x+4, y-36)
	errLabel.setText(err)
	errLabel.show()
}

func clearError() {
	errLabel.hide()
}

func main() {
	client = api.NewClient(apiURL)

	loadGraph()
	for c := range graph.Channels {
		c.makeElements()
	}
	for _, n := range graph.Nodes {
		n.makeElements()
	}

	errLabel = newTextBox("", errTextStyle, errRectStyle, 0, 0, 0, 32)
	errLabel.hide()

	diagramSVG.Call("addEventListener", "mousemove", mouseMove)
	diagramSVG.Call("addEventListener", "mouseup", mouseUp)

	saveGraphPropsLink.Call("addEventListener", "click", graph.saveProperties)
}

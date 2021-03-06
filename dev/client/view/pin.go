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

package view

import (
	"errors"

	"github.com/google/shenzhen-go/dev/dom"
	pb "github.com/google/shenzhen-go/dev/proto/js"
	"golang.org/x/net/context"
)

const (
	nametagRectStyle = "fill: #efe; stroke: #353; stroke-width:1"
	nametagTextStyle = "font-family:Go; font-size:16; user-select:none; pointer-events:none"
)

// Pin represents a node pin visually, and has enough information to know
// if it is validly connected.
type Pin struct {
	Name, Type string

	input bool     // am I an input?
	node  *Node    // owner.
	ch    *Channel // attached to this channel, is often nil

	nametag *textBox // Hello, my name is ...

	circ dom.Element // my main representation
	l    dom.Element // attached line; x1, y1 = x, y; x2, y2 = ch.tx, ch.ty.
	x, y float64     // computed, not relative to node
	c    dom.Element // circle, when dragging from a pin
}

// Save time by checking whether a potential connection can succeeds.
func (p *Pin) checkConnectionTo(q Point) error {
	switch q := q.(type) {
	case *Pin:
		if q.Type != p.Type {
			return errors.New("mismatching types [" + p.Type + " != " + q.Type + "]")
		}
		if q.ch != nil {
			return p.checkConnectionTo(q.ch)
		}

		// Prevent mistakes by ensuring that there is at least one input
		// and one output per channel, and they connect separate goroutines.
		if p.input == q.input {
			return errors.New("both pins have the same direction")
		}
		if p.node == q.node {
			return errors.New("both pins are on the same goroutine")
		}

	case *Channel:
		if q.channel.Type != p.Type {
			return errors.New("mismatching types [" + p.Type + " != " + q.channel.Type + "]")
		}
		same := true
		for r := range q.Pins {
			if r.input != p.input {
				same = false
				break
			}
		}
		if same {
			return errors.New("must connect at least one input and one output")
		}
	}
	return nil
}

func (p *Pin) connectTo(q Point) {
	switch q := q.(type) {
	case *Pin:
		if p.ch != nil && p.ch != q.ch {
			p.disconnect()
		}
		if q.ch != nil {
			p.connectTo(q.ch)
			return
		}

		// Create a new channel to connect to
		ch := p.node.view.createChannel(p, q)
		ch.reposition(nil)
		p.ch, q.ch = ch, ch
		p.node.view.graph.Channels[ch.channel.Name] = ch
		q.l.Show()

	case *Channel:
		if p.ch != nil && p.ch != q {
			p.disconnect()
		}

		p.ch = q
		q.Pins[p] = struct{}{}
		q.reposition(nil)
	}
	return
}

func (p *Pin) reallyConnect() {
	// Attach to the existing channel
	if _, err := p.node.view.client.ConnectPin(context.Background(), &pb.ConnectPinRequest{
		Graph:   p.node.view.graph.gc.Graph().FilePath,
		Node:    p.node.node.Name,
		Pin:     p.Name,
		Channel: p.ch.channel.Name,
	}); err != nil {
		p.node.view.diagram.setError("Couldn't connect: "+err.Error(), 0, 0)
	}
}

func (p *Pin) disconnect() {
	if p.ch == nil {
		return
	}
	go p.reallyDisconnect()
	delete(p.ch.Pins, p)
	p.ch.setColour(normalColour)
	p.ch.reposition(nil)
	if len(p.ch.Pins) < 2 {
		// Delete the channel
		for q := range p.ch.Pins {
			q.ch = nil
		}
		delete(p.node.view.graph.Channels, p.ch.channel.Name)
	}
	p.ch = nil
}

func (p *Pin) reallyDisconnect() {
	if _, err := p.node.view.client.DisconnectPin(context.Background(), &pb.DisconnectPinRequest{
		Graph: p.node.view.graph.gc.Graph().FilePath,
		Node:  p.node.node.Name,
		Pin:   p.Name,
	}); err != nil {
		p.node.view.diagram.setError("Couldn't disconnect: "+err.Error(), 0, 0)
	}
}

func (p *Pin) setPos(rx, ry float64) {
	p.circ.
		SetAttribute("cx", rx).
		SetAttribute("cy", ry)
	p.x, p.y = rx+p.node.node.X, ry+p.node.node.Y
	if p.l != nil {
		p.l.
			SetAttribute("x1", p.x).
			SetAttribute("y1", p.y)
	}
	if p.ch != nil {
		p.ch.reposition(nil)
		p.ch.commit()
	}
}

// Pt returns the diagram coordinate of the pin, for nearest-neighbor purposes.
func (p *Pin) Pt() (x, y float64) { return p.x, p.y }

func (p *Pin) String() string { return p.node.node.Name + "." + p.Name }

func (p *Pin) dragStart(e dom.Object) {
	// If the pin is attached to something, detach and drag from that instead.
	if ch := p.ch; ch != nil {
		p.disconnect()
		p.l.Hide()
		if len(ch.Pins) > 1 {
			ch.dragStart(e)
			return
		}
		for q := range ch.Pins {
			q.dragStart(e)
			return
		}
	}
	p.node.view.diagram.dragItem = p

	p.circ.SetAttribute("fill", errorColour)

	x, y := p.node.view.diagram.cursorPos(e)
	p.l.SetAttribute("x2", x).
		SetAttribute("y2", y).
		SetAttribute("stroke", errorColour).
		Show()
	p.c.SetAttribute("cx", x).
		SetAttribute("cy", y).
		SetAttribute("stroke", errorColour).
		Show()
}

func (p *Pin) drag(e dom.Object) {
	x, y := p.node.view.diagram.cursorPos(e)
	defer func() {
		p.l.SetAttribute("x2", x).SetAttribute("y2", y)
		p.c.SetAttribute("cx", x).SetAttribute("cy", y)
	}()
	d, q := p.node.view.graph.nearestPoint(x, y)

	noSnap := func() {
		if p.ch != nil {
			p.ch.setColour(normalColour)
			p.disconnect()
		}

		p.circ.SetAttribute("fill", errorColour)
		p.l.SetAttribute("stroke", errorColour)
		p.c.SetAttribute("stroke", errorColour).Show()
	}

	// Don't connect P to itself, don't connect if nearest is far away.
	if p == q || d >= snapQuad {
		p.node.view.diagram.clearError()
		noSnap()
		return
	}

	if err := p.checkConnectionTo(q); err != nil {
		p.node.view.diagram.setError("Can't connect: "+err.Error(), x, y)
		noSnap()
		return
	}

	// Make the connection.
	p.connectTo(q)

	// Snap to q.ch, or q if q is a channel. Visual.
	switch q := q.(type) {
	case *Pin:
		x, y = q.ch.tx, q.ch.ty
	case *Channel:
		x, y = q.tx, q.ty
	}

	// Valid snap - ensure the colour is active.
	p.node.view.diagram.clearError()
	p.ch.setColour(activeColour)
	p.c.Hide()
}

func (p *Pin) drop(e dom.Object) {
	p.node.view.diagram.clearError()
	p.circ.SetAttribute("fill", normalColour)
	p.c.Hide()
	if p.ch == nil {
		p.l.Hide()
		go p.reallyDisconnect()
		return
	}
	if p.ch.created {
		go p.reallyConnect()
	}
	p.ch.setColour(normalColour)
	p.ch.commit()
}

func (p *Pin) mouseEnter(dom.Object) {
	x, y := p.x-p.node.node.X, p.y-p.node.node.Y
	if p.input {
		y -= 38
	} else {
		y += 8
	}
	p.nametag.moveTo(x, y)
	p.nametag.show()
}

func (p *Pin) mouseLeave(dom.Object) {
	p.nametag.hide()
}

func (p *Pin) makeElements(n *Node) dom.Element {
	p.node = n
	p.circ = n.view.doc.MakeSVGElement("circle").
		SetAttribute("r", pinRadius).
		SetAttribute("fill", normalColour).
		AddEventListener("mousedown", p.dragStart).
		AddEventListener("mouseenter", p.mouseEnter).
		AddEventListener("mouseleave", p.mouseLeave)

	// Line
	p.l = n.view.doc.MakeSVGElement("line").
		SetAttribute("stroke-width", lineWidth).
		Hide()

	// Another circ
	p.c = n.view.doc.MakeSVGElement("circle").
		SetAttribute("r", pinRadius).
		SetAttribute("fill", "transparent").
		SetAttribute("stroke-width", lineWidth).
		Hide()

	n.view.diagram.AddChildren(p.l, p.c)

	// Nametag
	p.nametag = p.node.view.newTextBox(p.Name+" ("+p.Type+")", nametagTextStyle, nametagRectStyle, 0, 0, 0, 30)
	p.node.box.AddChildren(p.nametag)
	p.nametag.computeWidth()
	p.nametag.hide()
	return p.circ
}

func (p *Pin) unmakeElements() {
	p.node.view.diagram.RemoveChildren(p.l, p.c)
	p.nametag.unmakeElements()
}

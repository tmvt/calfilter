package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"sync"
)

type CalFilter struct {
	scanner     *bufio.Scanner
	buffer      *bytes.Buffer
	eventBuffer *bytes.Buffer
	inEvent     bool
	m           sync.Mutex
}

type Prop struct {
	Name  string
	Value string
}

type Event struct {
	Props map[string]Prop
	r     *bytes.Reader
}

func NewParser(r io.Reader) *CalFilter {
	return &CalFilter{
		scanner:     bufio.NewScanner(r),
		buffer:      new(bytes.Buffer),
		eventBuffer: new(bytes.Buffer),
	}
}

func (cf *CalFilter) Parse(writer io.Writer, rules []RuleGroup) (int64, error) {
	cf.m.Lock()
	defer cf.m.Unlock()
	log.Printf("Parsing started")
	// Scan through calendar file
	for cf.scanner.Scan() {
		cf.readLine()
		// If a VEvent has been fully traversed
		if cf.eventBuffer.Len() > 0 && cf.inEvent == false {
			event, err := cf.parseEvent()
			if err != nil {
				return 0, err
			}

			if ok, err := event.checkBlacklist(rules); ok && err == nil {
				// Add event to main buffer if filter does not apply
				for {
					b, err := event.r.ReadByte()
					if err == io.EOF {
						break
					} else if err != nil {
						return 0, err
					}
					cf.buffer.WriteByte(b)
				}
			} else if err != nil {
				return 0, err
			}
		}
	}
	log.Printf("Parsing finished")
	n, err := cf.buffer.WriteTo(writer)
	if err != nil {
		return n, err
	}
	return n, nil
}

func (cf *CalFilter) readLine() {
	s := cf.scanner.Text()

	var buf *bytes.Buffer
	if cf.inEvent {
		buf = cf.eventBuffer
	} else {
		buf = cf.buffer
	}

	if s == "BEGIN:VEVENT" {
		cf.inEvent = true
		buf = cf.eventBuffer
	} else if s == "END:VEVENT" {
		cf.inEvent = false
	}

	_, err := buf.WriteString(fmt.Sprintf("%v\n", s))
	if err != nil {
		panic(err)
	}
}

func (cf *CalFilter) parseEvent() (Event, error) {
	event := Event{
		Props: map[string]Prop{},
		r:     bytes.NewReader(cf.eventBuffer.Bytes()),
	}

	for {
		var sb strings.Builder

		s, err := cf.readEventLine()
		if err != nil {
			return Event{}, err
		}
		sb.WriteString(s)

		// Check for continuing line
		for {
			r, _, err := cf.eventBuffer.ReadRune()
			if err == io.EOF {
				break
			} else if err != nil {
				return Event{}, err
			}

			if r != ' ' && r != '\t' {
				err := cf.eventBuffer.UnreadRune()
				if err != nil {
					return Event{}, err
				}
				break
			}

			s, err := cf.readEventLine()
			if err != nil {
				return Event{}, err
			}
			sb.WriteString(s)
		}

		vals := strings.SplitN(sb.String(), ":", 2)
		if len(vals) != 2 {
			return Event{}, fmt.Errorf("invalid event prop")
		}
		prop := Prop{
			Name:  vals[0],
			Value: vals[1],
		}
		event.Props[prop.Name] = prop

		if prop.Name == "END" {
			break
		}
	}
	return event, nil
}

func (cf *CalFilter) readEventLine() (string, error) {
	s, err := cf.eventBuffer.ReadString('\n')
	s = strings.TrimRight(s, "\r\n")
	if err == io.EOF && len(s) > 0 {
		err = nil
	}
	return s, err
}

func (e *Event) checkBlacklist(ruleGroups []RuleGroup) (bool, error) {
	for _, ruleGroup := range ruleGroups {
		mode := ruleGroup.Mode
		if ruleGroup.Rules == nil {
			log.Println("Rule group does not contain any ruleGroups!")
			continue
		}

		matches := 0
		nonMatches := 0
		for k, rule := range ruleGroup.Rules {
			// TODO Move rule compilation for faster and more efficient parsing
			f, err := regexp.Compile(rule)
			if err != nil {
				return true, fmt.Errorf("rule with filter %q could not be compiled (%q)", rule, err)
			}
			p, ok := e.Props[k]
			if !ok {
				return true, fmt.Errorf("rule references unknown event property: %q", k)
			}

			if f.MatchString(p.Value) {
				matches++
			} else {
				nonMatches++
			}
		}

		switch mode {
		case ruleAnd:
			if nonMatches == 0 {
				log.Printf("Found match! Removed event %v\n", e.Props)
				return false, nil
			}
		case ruleOr:
			if matches > 0 {
				log.Printf("Found match! Removed event %v\n", e.Props)
				return false, nil
			}
		}

	}

	return true, nil
}

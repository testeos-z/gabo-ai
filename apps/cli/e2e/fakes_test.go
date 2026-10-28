//go:build e2e

package e2e

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
)

// The fakes keep their state in JSON files, so a second cycle run (the resume
// after an interruption) sees only what the first one wrote.

// issue is one task of the fake tracker (the tasks slot).
type issue struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	State     string   `json:"state"` // open | closed
	Labels    []string `json:"labels"`
	BlockedBy []int    `json:"blocked_by"`
	Comments  []string `json:"comments"`
}

func (i issue) id() string { return strings.SplitN(i.Title, ":", 2)[0] }

// tracker is the fake tasks adapter. drop names a write ("claim 2") that is
// lost once, so the re-read (R24) has something to catch.
type tracker struct {
	path string
	drop map[string]bool
}

func (t *tracker) load() ([]issue, error) {
	b, err := os.ReadFile(t.path)
	if err != nil {
		return nil, err
	}
	var all []issue
	return all, json.Unmarshal(b, &all)
}

func (t *tracker) save(all []issue) error {
	b, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.path, b, 0o600)
}

// readAll is the contract op "read unblocked" input: every task.
func (t *tracker) readAll() ([]issue, error) { return t.load() }

// readOne is the contract op "read one".
func (t *tracker) readOne(n int) (issue, error) {
	all, err := t.load()
	if err != nil {
		return issue{}, err
	}
	for _, i := range all {
		if i.Number == n {
			return i, nil
		}
	}
	return issue{}, errors.New("no such issue")
}

func (t *tracker) write(op string, n int, f func(*issue)) error {
	key := op + " " + itoa(n)
	if t.drop[key] {
		delete(t.drop, key)
		return nil // accepted, never stored
	}
	all, err := t.load()
	if err != nil {
		return err
	}
	for k := range all {
		if all[k].Number == n {
			f(&all[k])
		}
	}
	return t.save(all)
}

func (t *tracker) claim(n int, agent, wave string) error {
	return t.write("claim", n, func(i *issue) {
		if !slices.Contains(i.Labels, "status:in-progress") {
			i.Labels = append(i.Labels, "status:in-progress")
		}
		i.Comments = append(i.Comments, "claimed by "+agent+" (wave "+wave+")")
	})
}

func (t *tracker) comment(n int, body string) error {
	return t.write("comment", n, func(i *issue) { i.Comments = append(i.Comments, body) })
}

// close is the contract op "change status" DONE.
func (t *tracker) close(n, pr int) error {
	return t.write("close", n, func(i *issue) {
		i.State = "closed"
		i.Labels = slices.DeleteFunc(i.Labels, func(l string) bool { return l == "status:in-progress" })
		i.Comments = append(i.Comments, "DONE via PR #"+itoa(pr))
	})
}

// memory is the fake memory adapter: topics of append-only entries.
type memory struct{ path string }

func (m *memory) load() (map[string][]string, error) {
	topics := map[string][]string{}
	b, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return topics, nil
	}
	if err != nil {
		return nil, err
	}
	return topics, json.Unmarshal(b, &topics)
}

func (m *memory) append(topic, entry string) error {
	topics, err := m.load()
	if err != nil {
		return err
	}
	topics[topic] = append(topics[topic], entry)
	b, err := json.MarshalIndent(topics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, b, 0o600)
}

func (m *memory) read(topic string) ([]string, error) {
	topics, err := m.load()
	return topics[topic], err
}

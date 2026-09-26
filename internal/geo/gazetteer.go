package geo

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"strings"
	"sync"
)

type Gazetteer struct {
	once  sync.Once
	index map[string][]Place
	path  string
	err   error
}

type Place struct {
	Name   string
	ASCII  string
	Alpha2 string
	Admin1 string
	Lat    float64
	Lon    float64
	Pop    int
}

func NewGazetteer(path string) *Gazetteer { return &Gazetteer{path: path} }

func (g *Gazetteer) Resolve(name, country string) (Place, bool) {
	g.load()
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return Place{}, false
	}
	candidates := g.index[name]
	if len(candidates) == 0 {
		return Place{}, false
	}
	var best Place
	var found bool
	for _, p := range candidates {
		if country != "" && p.Alpha2 != country {
			continue
		}
		if !found || p.Pop > best.Pop {
			best = p
			found = true
		}
	}
	return best, found
}

func (g *Gazetteer) Size() int {
	g.load()
	return len(g.index)
}

func (g *Gazetteer) load() {
	g.once.Do(func() {
		if g.path == "" {
			g.index = map[string][]Place{}
			return
		}
		if _, err := os.Stat(g.path); err != nil {
			g.index = map[string][]Place{}
			return
		}
		g.err = g.parse(g.path)
		if g.index == nil {
			g.index = map[string][]Place{}
		}
	})
}

func (g *Gazetteer) parse(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var r *bufio.Scanner
	if strings.HasSuffix(path, ".gz") {
		zr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer zr.Close()
		r = bufio.NewScanner(zr)
	} else {
		r = bufio.NewScanner(f)
	}
	r.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	g.index = make(map[string][]Place, 1<<16)

	for r.Scan() {
		fields := strings.Split(r.Text(), "\t")
		if len(fields) < 15 {
			continue
		}
		p := Place{
			Name:   fields[1],
			ASCII:  fields[2],
			Alpha2: fields[8],
			Admin1: fields[10],
		}
		_, _ = fmt.Sscanf(fields[4], "%f", &p.Lat)
		_, _ = fmt.Sscanf(fields[5], "%f", &p.Lon)
		_, _ = fmt.Sscanf(fields[14], "%d", &p.Pop)

		for _, key := range []string{strings.ToLower(p.Name), strings.ToLower(p.ASCII)} {
			if key == "" {
				continue
			}
			g.index[key] = append(g.index[key], p)
		}
	}
	return r.Err()
}

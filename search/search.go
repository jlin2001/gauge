// Copyright 2015 ThoughtWorks, Inc.

// This file is part of Gauge.

// Gauge is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Gauge is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with Gauge.  If not, see <http://www.gnu.org/licenses/>.

package search

import (
	"path/filepath"

	"sync"

	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/mapping"
	"github.com/getgauge/gauge/config"
	"github.com/getgauge/gauge/gauge"
	"github.com/getgauge/gauge/logger"
)

const (
	dotGauge  = ".gauge"
	indexFile = "gauge.idx"
)

type specDoc struct {
	Id      string
	Heading string
	Context []string
	Comment []string
	Tag     []string
}

type scenarioDoc struct {
	Id      string
	Heading string
	Step    []string
	Comment []string
	Tag     []string
}

func (d *specDoc) Type() string {
	return "spec"
}

func (d *scenarioDoc) Type() string {
	return "scenario"
}

func newSpecDoc(s *gauge.Specification) *specDoc {
	id, err := filepath.Rel(config.ProjectRoot, s.FileName)
	if err != nil {
		logger.Errorf("Unable to get relative path for %s. %s", s.FileName, err)
		return nil
	}

	specDoc := &specDoc{
		Id:      id,
		Heading: s.Heading.Value,
		Context: make([]string, 0),
		Comment: make([]string, 0),
	}

	if s.Tags != nil {
		specDoc.Tag = s.Tags.Values
	}

	for _, step := range s.Contexts {
		fmt.Println(step.Value)
		specDoc.Context = append(specDoc.Context, step.Value)
	}

	for _, comment := range s.Comments {
		specDoc.Comment = append(specDoc.Comment, comment.Value)
	}

	return specDoc
}

func newScenarioDoc(scn *gauge.Scenario, filename string) *scenarioDoc {
	scnID := fmt.Sprintf("%s:%d", filename, scn.Heading.LineNo)
	scnDoc := &scenarioDoc{
		Id:      scnID,
		Heading: scn.Heading.Value,
		Step:    make([]string, 0),
		Comment: make([]string, 0),
	}
	if scn.Tags != nil {
		scnDoc.Tag = scn.Tags.Values
	}

	for _, comment := range scn.Comments {
		scnDoc.Comment = append(scnDoc.Comment, comment.Value)
	}

	for _, step := range scn.Steps {
		scnDoc.Step = append(scnDoc.Step, step.Value)
	}
	return scnDoc
}

func Search(q string) {
	indexPath := filepath.Join(config.ProjectRoot, dotGauge, indexFile)
	index, err := createOrOpenIndex(indexPath)

	if err != nil {
		logger.Warningf("Unable to open index : %s. %s", indexPath, err)
	}

	query := bleve.NewMatchQuery(q)
	tagsFacet := bleve.NewFacetRequest("Tag", 5)
	search := bleve.NewSearchRequest(query)
	search.AddFacet("tags", tagsFacet)
	search.Highlight = bleve.NewHighlight()
	searchResults, err := index.Search(search)

	if err != nil {
		fmt.Printf("Error searching : %s\n", err)
	} else {
		fmt.Println(searchResults)
	}
}

// Initialize sets up the search index.
func Initialize(specs *gauge.SpecCollection) {
	indexPath := filepath.Join(config.ProjectRoot, dotGauge, indexFile)
	gaugeIndex, err := createOrOpenIndex(indexPath)

	if err != nil {
		logger.Warningf("Unable to open index : %s. %s", indexPath, err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(specs.Size())
	logger.Infof("Indexing %d specs", specs.Size())
	for _, spec := range specs.Specs() {
		logger.Infof("Indexing %s", spec.FileName)
		go func(s *gauge.Specification) {
			d := newSpecDoc(s)
			gaugeIndex.Index(d.Id, d)
			wg.Done()
		}(spec)
		wg.Add(len(spec.Scenarios))
		for _, scn := range spec.Scenarios {
			go func(s *gauge.Scenario, f string) {
				d := newScenarioDoc(s, f)
				gaugeIndex.Index(d.Id, d)
				wg.Done()
			}(scn, spec.FileName)
		}
	}
	wg.Wait()

	json, _ := gaugeIndex.Stats().MarshalJSON()
	logger.Infof(string(json))
	gaugeIndex.Close()
}

func createOrOpenIndex(indexPath string) (bleve.Index, error) {
	gaugeIndex, err := bleve.Open(indexPath)

	if err == bleve.ErrorIndexPathDoesNotExist {
		gaugeIndex, err = bleve.New(indexPath, buildIndexMapping())
		if err != nil {
			return nil, err
		}
	}
	return gaugeIndex, nil
}

func buildIndexMapping() mapping.IndexMapping {
	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = standard.Name

	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	indexMapping := bleve.NewIndexMapping()

	scenarioMapping := bleve.NewDocumentMapping()
	scenarioMapping.AddFieldMappingsAt("Heading", englishTextFieldMapping)
	scenarioMapping.AddFieldMappingsAt("Tag", keywordFieldMapping)

	specMapping := bleve.NewDocumentMapping()
	specMapping.AddFieldMappingsAt("Heading", englishTextFieldMapping)
	specMapping.AddFieldMappingsAt("Tag", keywordFieldMapping)

	// specMapping.AddSubDocumentMapping("scenarios", scenarioMapping)
	indexMapping.AddDocumentMapping("spec", specMapping)
	indexMapping.AddDocumentMapping("scenario", scenarioMapping)

	return indexMapping
}

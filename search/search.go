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
	Id           string
	heading      string
	contextSteps []string
	comments     []string
	tags         []string
	scenarios    []*scenarioDoc
}

type scenarioDoc struct {
	id       string
	heading  string
	steps    []string
	comments []string
	tags     []string
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
		Id:           id,
		heading:      s.Heading.Value,
		contextSteps: make([]string, 0),
		comments:     make([]string, 0),
	}

	if s.Tags != nil {
		specDoc.tags = s.Tags.Values
	}

	for _, step := range s.Contexts {
		specDoc.contextSteps = append(specDoc.contextSteps, step.Value)
	}

	for _, comment := range s.Comments {
		specDoc.comments = append(specDoc.comments, comment.Value)
	}

	// for _, scn := range s.Scenarios {
	// 	scnID := fmt.Sprintf("%s:%d", id, scn.Heading.LineNo)
	// 	scnDoc := &scenarioDoc{
	// 		id:       scnID,
	// 		heading:  scn.Heading.Value,
	// 		steps:    make([]string, 0),
	// 		comments: make([]string, 0),
	// 	}
	// 	if scn.Tags != nil {
	// 		scnDoc.tags = scn.Tags.Values
	// 	}

	// 	for _, comment := range scn.Comments {
	// 		scnDoc.comments = append(scnDoc.comments, comment.Value)
	// 	}

	// 	for _, step := range scn.Steps {
	// 		scnDoc.steps = append(scnDoc.steps, step.Value)
	// 	}

	// 	specDoc.scenarios = append(specDoc.scenarios, scnDoc)
	// }

	return specDoc
}

func Search(q string) {
	indexPath := filepath.Join(config.ProjectRoot, dotGauge, indexFile)
	index, err := createOrOpenIndex(indexPath)

	if err != nil {
		logger.Warningf("Unable to open index : %s. %s", indexPath, err)
	}

	query := bleve.NewMatchQuery(q)
	search := bleve.NewSearchRequest(query)
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
			gaugeIndex.Index("id", newSpecDoc(s))
			wg.Done()
		}(spec)
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

	// scenarioMapping := bleve.NewDocumentStaticMapping()
	// scenarioMapping.AddFieldMappingsAt("heading", englishTextFieldMapping)
	// scenarioMapping.AddFieldMappingsAt("tags", keywordFieldMapping)

	specMapping := bleve.NewDocumentStaticMapping()
	specMapping.AddFieldMappingsAt("heading", englishTextFieldMapping)
	specMapping.AddFieldMappingsAt("tags", keywordFieldMapping)

	// specMapping.AddSubDocumentMapping("scenarios", scenarioMapping)
	indexMapping.AddDocumentMapping("spec", specMapping)

	return indexMapping
}

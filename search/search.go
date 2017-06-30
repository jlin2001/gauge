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

func Search(q string) {
	indexPath := filepath.Join(config.ProjectRoot, dotGauge, indexFile)
	index, err := createOrOpenIndex(indexPath)

	if err != nil {
		logger.Warning("Unable to open index : %s. %s", indexPath, err)
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
		logger.Warning("Unable to open index : %s. %s", indexPath, err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(specs.Size())
	logger.Info("Indexing %d specs", specs.Size())
	for _, spec := range specs.Specs() {
		logger.Info("Indexing %s", spec.FileName)
		go indexSpec(gaugeIndex, spec, wg)
	}
	wg.Wait()

	json, _ := gaugeIndex.Stats().MarshalJSON()
	logger.Info(string(json))
	gaugeIndex.Close()
}

func indexSpec(index bleve.Index, spec *gauge.Specification, wg *sync.WaitGroup) {
	specID, err := filepath.Rel(config.ProjectRoot, spec.FileName)
	if err != nil {
		logger.Errorf("Unable to get relative path for %s. %s", spec.FileName, err)
	}

	err = index.Index(specID, spec)
	if err != nil {
		logger.Errorf("Unable to index %s. %s", spec.FileName, err)
	}

	for _, scn := range spec.Scenarios {
		scnID := fmt.Sprintf("%s:%d", specID, scn.Heading.LineNo)
		logger.Info("Indexing scenario %s", scnID)
		err = index.Index(scnID, scn)
		if err != nil {
			logger.Errorf("Unable to index %s. %s", scnID, err)
		}
	}
	wg.Done()
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

	// a generic reusable mapping for keyword text
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	indexMapping := bleve.NewIndexMapping()

	scenarioMapping := bleve.NewDocumentStaticMapping()
	scenarioMapping.AddFieldMappingsAt("Heading.Value", englishTextFieldMapping)
	scenarioMapping.AddFieldMappingsAt("Tags", keywordFieldMapping)

	specMapping := bleve.NewDocumentStaticMapping()
	specMapping.AddFieldMappingsAt("Heading.Value", englishTextFieldMapping)
	specMapping.AddFieldMappingsAt("Tags", keywordFieldMapping)

	// specMapping.AddSubDocumentMapping("Scenarios", scenarioMapping)

	indexMapping.AddDocumentMapping("Specification", specMapping)
	indexMapping.AddDocumentMapping("Scenario", scenarioMapping)

	return indexMapping
}

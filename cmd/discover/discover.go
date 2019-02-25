package main

import (
	"encoding/csv"
	"io"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/dimdiden/oo"
	"github.com/olekukonko/tablewriter"
)

type pair struct {
	target   *oo.Asset
	similars *oo.Similars
}

type pairs []pair

// Len, Swap, Less are needed to satisfy Sort interface
func (p pairs) Len() int {
	return len(p)
}
func (p pairs) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p pairs) Less(i, j int) bool {
	return p[i].target.UpdatedAt > p[j].target.UpdatedAt
}

func (p pairs) renderAggregateResult(writer io.Writer) error {
	w := csv.NewWriter(writer)
	header := []string{"Asset", "Recommendation", "Reason"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, pr := range p {
		for _, similar := range pr.similars.Assets {
			record := []string{
				pr.target.EmbedCode,
				similar.EmbedCode,
				similar.Reason,
			}
			if err := w.Write(record); err != nil {
				return err
			}
		}
	}
	// Write any buffered data to the underlying writer (standard output).
	w.Flush()

	if err := w.Error(); err != nil {
		return err
	}
	return nil
}

func (p pairs) renderCommonResult(writer io.Writer, numRows int) error {
	table := tablewriter.NewWriter(writer)
	table.SetHeader(buildTableHeader("Asset", numRows))
	table.SetRowLine(true)
	table.AppendBulk(formTableData(p))
	table.Render()
	return nil
}

func buildTableHeader(first string, numRows int) []string {
	header := []string{first}
	for i := 1; i <= numRows; i++ {
		header = append(header, "Reccomendation "+strconv.Itoa(i))
	}
	return header
}

func formTableData(p pairs) [][]string {
	var tableData [][]string
	for _, pr := range p {
		tableRow := []string{pr.target.EmbedCode + "\n" + pr.target.UpdatedAt}

		var results []string
		for _, similar := range pr.similars.Assets {
			results = append(results, similar.EmbedCode+"\n"+similar.Reason+"\n"+similar.CreatedAt)
		}
		tableRow = append(tableRow, results...)

		tableData = append(tableData, tableRow)
	}
	return tableData
}

func loadDataFromCSV(path string) ([]*oo.Asset, error) {
	var targets []*oo.Asset

	inputCSV, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer inputCSV.Close()

	csvReader := csv.NewReader(inputCSV)
	lines, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	for i, line := range lines {
		if i == 0 {
			continue
		}
		target := &oo.Asset{
			UpdatedAt: line[0],
			EmbedCode: line[1],
			Name:      line[2],
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func getPairs(targets []*oo.Asset, client *oo.Client, v url.Values) (pairs, error) {
	var data pairs
	pairChan := make(chan pair, 1)
	errChan := make(chan error, 1)
	doneChan := make(chan bool, 1)
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(target *oo.Asset, client *oo.Client, v url.Values, wg *sync.WaitGroup, pairChan chan pair, errChan chan error) {
			defer wg.Done()

			similars, err := oo.GetNewSimilars(client, target.EmbedCode, v)
			if err != nil {
				errChan <- err
				return
			}
			sort.Sort(similars)
			pairChan <- pair{target, similars}
		}(target, client, v, &wg, pairChan, errChan)
	}
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	for {
		select {
		case <-doneChan:
			sort.Sort(data)
			return data, nil
		case err := <-errChan:
			return nil, err
		case pair := <-pairChan:
			data = append(data, pair)
		}
	}
}

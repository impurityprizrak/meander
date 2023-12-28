package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

/*
The Backlog is the node entrypoint to the local ElasticSearch DB, that contains
all the clients and chain data.

It's a implementation of elasticsearch.Client that has some utilities to read and insert
new documents or indices into the node database.

The backlog is flexible and can be created anytime. To create a new backlog, you must to
call the `NewBacklog` method. If you need to connect to an external database, just pass its
address as `string` argument. If nothing is passed, the function will try to connect to the
default address `http://localhost:9200`
*/
type Backlog struct {
	*elasticsearch.Client
}

func NewBacklog(address ...string) *Backlog {
	const BaseURI string = "http://localhost:9200"

	if len(address) == 0 {
		address = append(address, BaseURI)
	}

	cfg := elasticsearch.Config{
		Addresses: []string{
			address[0],
		},
	}

	es, err := elasticsearch.NewClient(cfg)

	if err != nil {
		log.Fatalf("Failed to create elasticsearch client: %s", err)
	}

	nodeStorage := Backlog{Client: es}
	return &nodeStorage
}

// This method creates the essential indices of the node backlog
func (b Backlog) Initialize() {
	indexes := []string{"peers", "clients", "transactions", "blockchain", "node", "cache"}

	for _, index := range indexes {
		err := b.IndexExists(index)

		if err != nil {
			err := b.CreateIndex(index)
			if err != nil {
				log.Fatalf("Failed to create index %s: %v", index, err)
			}
		} else {
			fmt.Printf("Index %s already exists\n", index)
		}
	}
}

// An util implementation of index existance verification process in ElasticSearch
func (b Backlog) IndexExists(index string) error {
	ctx := context.Background()

	req := esapi.IndicesGetRequest{
		Index: []string{index},
	}

	res, err := req.Do(ctx, b)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to get index: %s", res.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode JSON response: %s", err)
	}

	return nil
}

// An util implementation of index creating process in ElasticSearch
func (b Backlog) CreateIndex(index string) error {
	ctx := context.Background()

	req := esapi.IndicesCreateRequest{
		Index: index,
	}

	res, err := req.Do(ctx, b)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	fmt.Printf("Index %s created\n", index)
	return nil
}

// An util implementation of document indexing process in ElasticSearch
func (b Backlog) IndexDocument(index, id string, document map[string]interface{}) error {
	ctx := context.Background()

	if _, err := b.GetDocument(index, id); err != nil {
		return b.UpdateDocument(index, id, document)
	}

	jsonDocument, err := json.Marshal(document)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewBuffer(jsonDocument),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, b)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index the document: %s", res.String())
	}

	return nil
}

// An util implementation of document updating process in ElasticSearch
func (b Backlog) UpdateDocument(index, id string, document map[string]interface{}) error {
	ctx := context.Background()

	jsonDocument, err := json.Marshal(map[string]interface{}{
		"doc": document,
	})

	if err != nil {
		return err
	}

	req := esapi.UpdateRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewBuffer(jsonDocument),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, b)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to update the document: %s", res.String())
	}

	return nil
}

// An util implementation of document listing process in ElasticSearch
func (b Backlog) ListDocuments(index string, uri ...string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	ctx := context.Background()

	req := esapi.SearchRequest{
		Index: []string{index},
	}

	res, err := req.Do(ctx, b)
	if err != nil {
		return results, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return results, fmt.Errorf("failed to list documents: %s", res.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return results, fmt.Errorf("failed to decode JSON response: %s", err)
	}

	hits := response["hits"].(map[string]interface{})["hits"].([]interface{})
	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})
		id := hitMap["_id"].(string)
		source := hitMap["_source"].(map[string]interface{})
		source["_id"] = id

		results = append(results, source)
	}

	return results, nil
}

// An util implementation of document text-based searching process in ElasticSearch
func (b Backlog) FindDocument(index, key, value string) (map[string]interface{}, error) {
	var document map[string]interface{}
	ctx := context.Background()

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				key: value,
			},
		},
	}
	jsonQuery, _ := json.Marshal(query)

	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  bytes.NewBuffer(jsonQuery),
	}

	res, err := req.Do(ctx, b)
	if err != nil {
		return document, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return document, fmt.Errorf("failed to find document: %s", res.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return document, fmt.Errorf("failed to decode JSON response: %s", err)
	}

	hits := response["hits"].(map[string]interface{})["hits"].([]interface{})
	if len(hits) > 0 {
		hitMap := hits[0].(map[string]interface{})
		id := hitMap["_id"]
		document = hitMap["_source"].(map[string]interface{})
		document["_id"] = id

		return document, nil
	} else {
		fmt.Println("No documents found")
	}

	return document, nil
}

// An util implementation of document finding by id process in ElasticSearch
func (b Backlog) GetDocument(index, id string) (map[string]interface{}, error) {
	var document map[string]interface{}
	ctx := context.Background()

	req := esapi.GetRequest{
		Index:      index,
		DocumentID: id,
	}

	res, err := req.Do(ctx, b)
	if err != nil {
		return document, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return document, fmt.Errorf("failed to get document: %s", res.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return document, fmt.Errorf("failed to decode JSON response: %s", err)
	}

	document = response["_source"].(map[string]interface{})
	return document, nil
}

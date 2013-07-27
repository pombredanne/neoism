// Copyright (c) 2012-2013 Jason McVetta.  This is Free Software, released under
// the terms of the GPL v3.  See http://www.gnu.org/copyleft/gpl.html for details.
// Resist intellectual serfdom - the ownership of ideas is akin to slavery.

package neo4j

import (
	"encoding/json"
	"github.com/jmcvetta/restclient"
)

// A CypherQuery is a statement in the Cypher query language, with optional
// parameters and result.  If Result value is supplied, result data will be
// unmarshalled into it when the query is executed. Result must be a pointer
// to a slice of structs - e.g. &[]someStruct{}.
type CypherQuery struct {
	Statement  string                 `json:"statement"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"-"`
	cr         cypherResult
}

// Columns returns the names, in order, of the columns returned for this query.
// Empty if query has not been executed.
func (cq *CypherQuery) Columns() []string {
	return cq.cr.Columns
}

// Unmarshall decodes result data into v, which must be a pointer to a slice of
// structs - e.g. &[]someStruct{}.  Struct fields are matched up with fields
// returned by the cypher query using the `json:"fieldName"` tag.
func (cq *CypherQuery) Unmarshall(v interface{}) error {
	// We do a round-trip thru the JSON marshaller.  A fairly simple way to
	// do type-safe unmarshalling, but perhaps not the most efficient solution.
	rs := make([]map[string]*json.RawMessage, len(cq.cr.Data))
	for rowNum, row := range cq.cr.Data {
		m := map[string]*json.RawMessage{}
		for colNum, col := range row {
			name := cq.cr.Columns[colNum]
			m[name] = col
		}
		rs[rowNum] = m
	}
	b, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		logPretty(err)
		return err
	}
	return json.Unmarshal(b, v)
}

type cypherRequest struct {
	Query      string                 `json:"query"`
	Parameters map[string]interface{} `json:"params"`
}

type cypherResult struct {
	Columns []string
	Data    [][]*json.RawMessage
}

// Cypher executes a db query written in the Cypher language.  Data returned
// from the db is used to populate `result`, which should be a pointer to a
// slice of structs.  TODO:  Or a pointer to a two-dimensional array of structs?
func (db *Database) Cypher(q *CypherQuery) error {
	cRes := cypherResult{}
	cReq := cypherRequest{
		Query:      q.Statement,
		Parameters: q.Parameters,
	}
	ne := new(neoError)
	rr := restclient.RequestResponse{
		Url:    db.HrefCypher,
		Method: "POST",
		Data:   &cReq,
		Result: &cRes,
		Error:  ne,
	}
	status, err := db.rc.Do(&rr)
	if err != nil {
		return err
	}
	if status != 200 {
		logPretty(rr)
		return BadResponse
	}
	q.cr = cRes
	if q.Result != nil {
		q.Unmarshall(q.Result)
	}
	return nil
}

// Copyright (c) 2012-2013 Jason McVetta.  This is Free Software, released under
// the terms of the GPL v3.  See http://www.gnu.org/copyleft/gpl.html for details.
// Resist intellectual serfdom - the ownership of ideas is akin to slavery.

package neoism

import (
	"strconv"
	"strings"
)

// CreateNode creates a Node in the database.
func (db *Database) CreateNode(p Props) (*Node, error) {
	n := Node{}
	n.Db = db
	resp, err := db.Session.Post(db.HrefNode, &p, &n)
	if err != nil || resp.Status() != 201 {
		logPretty(resp.Status())
		ne := NeoError{}
		resp.Unmarshal(&ne)
		logPretty(ne)
		return &n, err
	}
	return &n, nil
}

// Node fetches a Node from the database
func (db *Database) Node(id int) (*Node, error) {
	uri := join(db.HrefNode, strconv.Itoa(id))
	return db.getNodeByUri(uri)
}

// getNodeByUri fetches a Node from the database based on its URI.
func (db *Database) getNodeByUri(uri string) (*Node, error) {
	n := Node{}
	n.Db = db
	resp, err := db.Session.Get(uri, nil, &n)
	if err != nil {
		return nil, err
	}
	status := resp.Status()
	switch {
	case status == 404:
		return &n, NotFound
	case status != 200 || n.HrefSelf == "":
		ne := NeoError{}
		resp.Unmarshal(&ne)
		logPretty(ne)
		return nil, ne
	}
	return &n, nil
}

// A Node is a node, with optional properties, in a graph.
type Node struct {
	entity
	HrefOutgoingRels      string                 `json:"outgoing_relationships"`
	HrefTraverse          string                 `json:"traverse"`
	HrefAllTypedRels      string                 `json:"all_typed_relationships"`
	HrefOutgoing          string                 `json:"outgoing_typed_relationships"`
	HrefIncomingRels      string                 `json:"incoming_relationships"`
	HrefCreateRel         string                 `json:"create_relationship"`
	HrefPagedTraverse     string                 `json:"paged_traverse"`
	HrefAllRels           string                 `json:"all_relationships"`
	HrefIncomingTypedRels string                 `json:"incoming_typed_relationships"`
	HrefLabels            string                 `json:"labels"`
	Data                  map[string]interface{} `json:"data"`
	Extensions            map[string]interface{} `json:"extensions"`
}

// Id gets the ID number of this Node.
func (n *Node) Id() int {
	l := len(n.Db.HrefNode)
	s := n.HrefSelf[l:]
	s = strings.Trim(s, "/")
	id, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return id
}

// getRels makes an api call to the supplied uri and returns a map
// keying relationship IDs to Rel objects.
func (n *Node) getRels(uri string, types ...string) (Rels, error) {
	if types != nil {
		fragment := strings.Join(types, "&")
		parts := []string{uri, fragment}
		uri = strings.Join(parts, "/")
	}
	rels := Rels{}
	resp, err := n.Db.Session.Get(uri, nil, &rels)
	if err != nil {
		return rels, err
	}
	if resp.Status() != 200 {
		ne := NeoError{}
		resp.Unmarshal(&ne)
		logPretty(ne)
		return rels, ne
	}
	return rels, nil // Success!
}

// Rels gets all Rels for this Node, optionally filtered by
// type, returning them as a map keyed on Rel ID.
func (n *Node) Relationships(types ...string) (Rels, error) {
	return n.getRels(n.HrefAllRels, types...)
}

// Incoming gets all incoming Rels for this Node.
func (n *Node) Incoming(types ...string) (Rels, error) {
	return n.getRels(n.HrefIncomingRels, types...)
}

// Outgoing gets all outgoing Rels for this Node.
func (n *Node) Outgoing(types ...string) (Rels, error) {
	return n.getRels(n.HrefOutgoingRels, types...)
}

// Relate creates a relationship of relType, with specified properties,
// from this Node to the node identified by destId.
func (n *Node) Relate(relType string, destId int, p Props) (*Relationship, error) {
	rel := Relationship{}
	rel.Db = n.Db
	srcUri := join(n.HrefSelf, "relationships")
	destUri := join(n.Db.HrefNode, strconv.Itoa(destId))
	content := map[string]interface{}{
		"to":   destUri,
		"type": relType,
	}
	if p != nil {
		content["data"] = &p
	}
	resp, err := n.Db.Session.Post(srcUri, content, &rel)
	if err != nil {
		return &rel, err
	}
	if resp.Status() != 201 {
		ne := NeoError{}
		resp.Unmarshal(&ne)
		logPretty(ne)
		return &rel, ne
	}
	return &rel, nil
}

// AddLabels adds one or more labels to a node.
func (n *Node) AddLabel(labels ...string) error {
	resp, err := n.Db.Session.Post(n.HrefLabels, labels, nil)
	if err != nil {
		return err
	}
	if resp.Status() == 404 {
		return NotFound
	}
	if resp.Status() != 204 {
		ne := NeoError{}
		resp.Unmarshal(&ne)
		return ne
	}
	return nil // Success
}

// Labels lists labels for a node.
func (n *Node) Labels() ([]string, error) {
	res := []string{}
	resp, err := n.Db.Session.Get(n.HrefLabels, nil, &res)
	if err != nil {
		return res, err
	}
	if resp.Status() == 404 {
		return res, NotFound
	}
	if resp.Status() != 200 {
		ne := NeoError{}
		resp.Unmarshal(&ne)
		return res, ne
	}
	return res, nil // Success
}

// RemoveLabel removes a label from a node.
func (n *Node) RemoveLabel(label string) error {
	uri := join(n.HrefLabels, label)
	resp, err := n.Db.Session.Delete(uri)
	if err != nil {
		return err
	}
	if resp.Status() == 404 {
		return NotFound
	}
	if resp.Status() != 204 {
		ne := NeoError{}
		resp.Unmarshal(&ne)
		return ne
	}
	return nil // Success
}

// SetLabels removes any labels currently on a node, and replaces them with the
// labels provided as argument.
func (n *Node) SetLabels(labels []string) error {
	resp, err := n.Db.Session.Put(n.HrefLabels, labels, nil)
	if err != nil {
		return err
	}
	if resp.Status() == 404 {
		return NotFound
	}
	if resp.Status() != 204 {
		ne := NeoError{}
		resp.Unmarshal(&ne)
		return ne
	}
	return nil // Success
}

// NodesByLabel gets all nodes with a given label.
func (db *Database) NodesByLabel(label string) ([]*Node, error) {
	uri := join(db.Url, "label", label, "nodes")
	res := []*Node{}
	resp, err := db.Session.Get(uri, nil, &res)
	if err != nil {
		return res, err
	}
	if resp.Status() == 404 {
		return res, NotFound
	}
	if resp.Status() != 200 {
		ne := NeoError{}
		resp.Unmarshal(&ne)
		return res, ne
	}
	for _, n := range res {
		n.Db = db
	}
	return res, nil // Success
}

// Labels lists all labels.
func (db *Database) Labels() ([]string, error) {
	uri := join(db.Url, "labels")
	labels := []string{}
	resp, err := db.Session.Get(uri, nil, &labels)
	if err != nil {
		return labels, err
	}
	if resp.Status() != 200 {
		ne := NeoError{}
		resp.Unmarshal(&ne)
		return labels, ne
	}
	return labels, nil
}

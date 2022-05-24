package graph

import (
	"encoding/json"
	"fmt"
	semver2 "github.com/blang/semver/v4"
	"log"
	"os"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/iterator"
	"gonum.org/v1/gonum/graph/simple"
)

// GraphNode is a node in an implicit graph.
type GraphNode struct {
	id        int64
	Neighbors []graph.Node
}

type VersionInfo struct {
	Timestamp    string            `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies"`
}

type PackageInfo struct {
	Name     string                 `json:"name"`
	Versions map[string]VersionInfo `json:"versions"`
}

// Type structure for nodes. Name and Version can be removed if we find we don't use them often enough
type nodeInfo struct {
	id        string
	Name      string
	Version   string
	Timestamp string
}

// NodeInfo constructs a nodeInfo structure and automatically fills the id. This might not be the proper way to
// do it in Go so feel free to change this to a more idiomatic version.
func NodeInfo(name, version, timestamp string) *nodeInfo {
	return &nodeInfo{
		id:        fmt.Sprintf("%s-%s", name, version),
		Name:      name,
		Version:   version,
		Timestamp: timestamp,
	}
}

// NewGraphNode returns a new GraphNode.
func NewGraphNode(id int64) *GraphNode {
	return &GraphNode{id: id}
}

// Node allows GraphNode to satisfy the graph.Graph interface.
func (g *GraphNode) Node(id int64) graph.Node {
	if id == g.id {
		return g
	}

	seen := map[int64]struct{}{g.id: {}}

	for _, n := range g.Neighbors {
		if n.ID() == id {
			return n
		}

		if gn, ok := n.(*GraphNode); ok {
			if gn.Has(seen, id) {
				return gn
			}
		}
	}

	return nil
}

func (g *GraphNode) Has(seen map[int64]struct{}, id int64) bool {

	for _, n := range g.Neighbors {
		if _, ok := seen[n.ID()]; ok {
			continue
		}

		seen[n.ID()] = struct{}{}
		if n.ID() == id {
			return true
		}

		if gn, ok := n.(*GraphNode); ok {
			if gn.Has(seen, id) {
				return true
			}
		}
	}

	return false
}

// Nodes allows GraphNode to satisfy the graph.Graph interface.
func (g *GraphNode) Nodes() graph.Nodes {
	nodes := []graph.Node{g}
	seen := map[int64]struct{}{g.id: {}}

	for _, n := range g.Neighbors {
		nodes = append(nodes, n)
		seen[n.ID()] = struct{}{}

		if gn, ok := n.(*GraphNode); ok {
			nodes = gn.nodes(nodes, seen)
		}
	}

	return iterator.NewOrderedNodes(nodes)
}

func (g *GraphNode) nodes(dst []graph.Node, seen map[int64]struct{}) []graph.Node {

	for _, n := range g.Neighbors {
		if _, ok := seen[n.ID()]; ok {
			continue
		}

		dst = append(dst, n)
		if gn, ok := n.(*GraphNode); ok {
			dst = gn.nodes(dst, seen)
		}
	}

	return dst
}

// From allows GraphNode to satisfy the graph.Graph interface.
func (g *GraphNode) From(id int64) graph.Nodes {
	if id == g.ID() {
		return iterator.NewOrderedNodes(g.Neighbors)
	}

	seen := map[int64]struct{}{g.id: {}}

	for _, n := range g.Neighbors {
		seen[n.ID()] = struct{}{}

		if gn, ok := n.(*GraphNode); ok {
			if result := gn.FindNeighbors(id, seen); result != nil {
				return iterator.NewOrderedNodes(result)
			}
		}
	}

	return nil
}

func (g *GraphNode) FindNeighbors(id int64, seen map[int64]struct{}) []graph.Node {
	if id == g.ID() {
		return g.Neighbors
	}

	for _, n := range g.Neighbors {
		if _, ok := seen[n.ID()]; ok {
			continue
		}
		seen[n.ID()] = struct{}{}

		if gn, ok := n.(*GraphNode); ok {
			if result := gn.FindNeighbors(id, seen); result != nil {
				return result
			}
		}
	}

	return nil
}

// HasEdgeBetween allows GraphNode to satisfy the graph.Graph interface.
func (g *GraphNode) HasEdgeBetween(uid, vid int64) bool {
	return g.EdgeBetween(uid, vid) != nil
}

// Edge allows GraphNode to satisfy the graph.Graph interface.
func (g *GraphNode) Edge(uid, vid int64) graph.Edge {
	return g.EdgeBetween(uid, vid)
}

// EdgeBetween allows GraphNode to satisfy the graph.Graph interface.
func (g *GraphNode) EdgeBetween(uid, vid int64) graph.Edge {
	if uid == g.id || vid == g.id {
		for _, n := range g.Neighbors {
			if n.ID() == uid || n.ID() == vid {
				return simple.Edge{F: g, T: n}
			}

		}
		return nil
	}

	seen := map[int64]struct{}{g.id: {}}

	for _, n := range g.Neighbors {
		seen[n.ID()] = struct{}{}
		if gn, ok := n.(*GraphNode); ok {
			if result := gn.edgeBetween(uid, vid, seen); result != nil {
				return result
			}
		}
	}

	return nil
}

func (g *GraphNode) edgeBetween(uid, vid int64, seen map[int64]struct{}) graph.Edge {
	if uid == g.id || vid == g.id {
		for _, n := range g.Neighbors {
			if n.ID() == uid || n.ID() == vid {
				return simple.Edge{F: g, T: n}
			}
		}
		return nil
	}

	for _, n := range g.Neighbors {
		if _, ok := seen[n.ID()]; ok {
			continue
		}

		seen[n.ID()] = struct{}{}
		if gn, ok := n.(*GraphNode); ok {
			if result := gn.edgeBetween(uid, vid, seen); result != nil {
				return result
			}
		}
	}

	return nil
}

// ID allows GraphNode to satisfy the graph.Node interface.
func (g *GraphNode) ID() int64 {
	return g.id
}

// AddMeighbor adds an edge between g and n.
func (g *GraphNode) AddNeighbor(n *GraphNode) {
	g.Neighbors = append(g.Neighbors, graph.Node(n))
}

func CreateMap(in *[]PackageInfo) *map[int64]nodeInfo {
	var id int64 = 0
	packagesInfo := *in
	m := make(map[int64]nodeInfo, len(packagesInfo))
	for _, packageInfo := range packagesInfo {
		for packageVersion, versionInfo := range packageInfo.Versions {
			m[id] = *NodeInfo(packageInfo.Name, packageVersion, versionInfo.Timestamp)
			id++
		}
	}
	return &m
}

func AddElementToMap(x PackageInfo, inputMap *map[int64]PackageInfo) {
	m := *inputMap
	m[int64(len(m))] = x
}

func CreateNameToIDMap(m *map[int64]nodeInfo) *map[string]int64 {
	newMap := make(map[string]int64, len(*m))
	for id, key := range *m {
		newMap[key.id] = id
	}
	return &newMap
}

func CreateNameToVersionMap(m *[]PackageInfo) *map[string][]string {
	newMap := make(map[string][]string, len(*m))
	for _, value := range *m {
		name := value.Name
		for k, _ := range value.Versions {
			newMap[name] = append(newMap[name], k)
		}
	}
	return &newMap
}

//func CreateHelperMap(m map[int64]PackageInfo) map[string]int64 {
//	n := make(map[string]int64)
//	for x, y := range m {
//
//	}
//	return n
//}

func CreateGraph(inputMap *map[int64]nodeInfo) *simple.DirectedGraph {
	m := *inputMap
	graph := simple.NewDirectedGraph()
	for x := range m {
		graph.AddNode(NewGraphNode(x))
	}
	return graph
}

//Function to write the simple graph to a dot file so it could be visualized with GraphViz
//TODO Find out how to add the labels to the nodes
func Visualization(graph *simple.DirectedGraph, name string) {
	result, _ := dot.Marshal(graph, name, "", "  ")

	file, err := os.Create(name + ".dot")

	if err != nil {
		log.Fatal("Error!", err)
	}
	defer file.Close()

	fmt.Fprintf(file, string(result))

}

// CreateEdges takes a graph, a list of packages and their dependencies and a map of package names to package IDs
// and creates directed edges between the dependent library and its dependencies.
func CreateEdges(graph *simple.DirectedGraph, inputList *[]PackageInfo, nameToIDMap *map[string]int64, nameToVersionMap *map[string][]string) {
	packagesInfo := *inputList
	nameToID := *nameToIDMap
	nameToVersion := *nameToVersionMap
	for id, packageInfo := range packagesInfo {
		for _, dependencyInfo := range packageInfo.Versions {
			for dependencyName, dependencyVersion := range dependencyInfo.Dependencies {
				c, err := semver2.ParseRange(dependencyVersion)
				if err != nil {
					panic(err)
				}
				for _, v := range nameToVersion[dependencyName] {
					newVersion, _ := semver2.Parse(v)
					if c(newVersion) {
						dependencyNameVersionString := fmt.Sprintf("%s-%s", dependencyName, v)
						dependencyNode := graph.Node(nameToID[dependencyNameVersionString])
						packageNode := graph.Node(int64(id))
						graph.SetEdge(simple.Edge{F: packageNode, T: dependencyNode})
					}
				}
			}
		}
	}
}

func ParseJSON(inPath string) *[]PackageInfo {
	// For NPM at least, about 2 million packages are expected, so we initialize so the array doesn't have to be re-allocated all the time
	const expectedAmount int = 2000000
	// An array for now since lists aren't type-safe, and they would overcomplicate things
	result := make([]PackageInfo, 0, expectedAmount)
	f, err := os.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}

	dec := json.NewDecoder(f)

	//Read opening bracket
	if _, err := dec.Token(); err != nil {
		log.Fatal(err)
	}

	for dec.More() {
		var packageInfo PackageInfo

		if err := dec.Decode(&packageInfo); err != nil {
			log.Fatal(err)
		}
		result = append(result, packageInfo)
	}

	//Read closing bracket
	if _, err := dec.Token(); err != nil {
		log.Fatal(err)
	}
	return &result
}

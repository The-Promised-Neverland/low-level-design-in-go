package arrangement

import "parking_management/models"

func BuildMovementOrder(moves []models.VehicleMove) models.MovementOrder {
	order := models.MovementOrder{
		Moves: make([][]models.VehicleMove, 0),
		Loop:  make([]bool, 0),
	}
	if len(moves) == 0 {
		return order
	}
	nodes := indexMovesByTicket(moves)
	graph, _ := buildDirectedGraph(nodes)
	components := disconnectedComponents(nodes, graph)
	for _, comp := range components {
		ordered, loop := topoSort(comp, graph, nodes)
		order.Moves = append(order.Moves, ordered)
		order.Loop = append(order.Loop, loop)
	}
	return order
}

func indexMovesByTicket(moves []models.VehicleMove) map[string]models.VehicleMove {
	nodes := make(map[string]models.VehicleMove, len(moves))
	for _, m := range moves {
		nodes[m.TicketID] = m
	}
	return nodes
}

func buildDirectedGraph(nodes map[string]models.VehicleMove) (graph map[string][]string, indegree map[string]int) {
	graph = make(map[string][]string, len(nodes))
	indegree = make(map[string]int, len(nodes))
	for id := range nodes {
		graph[id] = nil
		indegree[id] = 0
	}
	for aID, a := range nodes {
		for bID, b := range nodes {
			if aID == bID {
				continue
			}
			if a.FromFloorNumber == b.ToFloorNumber {
				graph[aID] = append(graph[aID], bID)
				indegree[bID]++
			}
		}
	}
	return graph, indegree
}

func disconnectedComponents(nodes map[string]models.VehicleMove, graph map[string][]string) [][]string {
	undirected := make(map[string][]string, len(nodes))
	for u, outs := range graph {
		for _, v := range outs {
			undirected[u] = append(undirected[u], v)
			undirected[v] = append(undirected[v], u)
		}
	}
	visited := make(map[string]bool, len(nodes))
	components := make([][]string, 0)
	for start := range nodes {
		if visited[start] {
			continue
		}
		components = append(components, findDisconnectedComp(start, undirected, visited))
	}
	return components
}

func findDisconnectedComp(start string, undirected map[string][]string, visited map[string]bool) []string {
	comp := make([]string, 0)
	stack := []string{start}
	visited[start] = true
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		comp = append(comp, id)
		for _, n := range undirected[id] {
			if !visited[n] {
				visited[n] = true
				stack = append(stack, n)
			}
		}
	}
	return comp
}

func topoSort(comp []string, graph map[string][]string, nodes map[string]models.VehicleMove) ([]models.VehicleMove, bool) {
	inComp := make(map[string]bool, len(comp))
	localIn := make(map[string]int, len(comp))
	for _, id := range comp {
		inComp[id] = true
		localIn[id] = 0
	}
	for _, id := range comp {
		for _, next := range graph[id] {
			if inComp[next] {
				localIn[next]++
			}
		}
	}
	queue := make([]string, 0)
	for _, id := range comp {
		if localIn[id] == 0 {
			queue = append(queue, id)
		}
	}
	ordered := make([]models.VehicleMove, 0, len(comp))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ordered = append(ordered, nodes[id])
		for _, next := range graph[id] {
			if !inComp[next] {
				continue
			}
			localIn[next]--
			if localIn[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if len(ordered) < len(comp) {
		all := make([]models.VehicleMove, 0, len(comp))
		for _, id := range comp {
			all = append(all, nodes[id])
		}
		return all, true
	}
	return ordered, false
}

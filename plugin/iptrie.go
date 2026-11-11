package main

import "net"

type trieNode struct {
	next [2]*trieNode
	leaf bool
}

type ipTrie struct {
	v4 *trieNode
	v6 *trieNode
}

func newIPTrie() *ipTrie {
	return &ipTrie{v4: &trieNode{}, v6: &trieNode{}}
}

func (tree *ipTrie) insert(addr net.IP, mask net.IPMask) {
	node := tree.v6
	if v4 := addr.To4(); v4 != nil {
		addr = v4
		node = tree.v4
	}
	ones, _ := mask.Size()
	for position := 0; position < ones; position++ {
		bit := (addr[position/8] >> (7 - position%8)) & 1
		if node.next[bit] == nil {
			node.next[bit] = &trieNode{}
		}
		node = node.next[bit]
		if node.leaf {
			return
		}
	}
	node.leaf = true
}

func (tree *ipTrie) search(addr net.IP) bool {
	node := tree.v6
	if v4 := addr.To4(); v4 != nil {
		addr = v4
		node = tree.v4
	}
	for position := 0; position < len(addr)*8; position++ {
		if node.leaf {
			return true
		}
		bit := (addr[position/8] >> (7 - position%8)) & 1
		node = node.next[bit]
		if node == nil {
			return false
		}
	}
	return node.leaf
}

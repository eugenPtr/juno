package trie

import (
	"errors"
	"fmt"

	"github.com/NethermindEth/juno/core/felt"
)

// https://github.com/starknet-io/starknet-p2p-specs/blob/main/p2p/proto/snapshot.proto#L6
type ProofNode struct {
	Binary *Binary
	Edge   *Edge
}

// Note: does not work for leaves
func (pn *ProofNode) Hash(hash hashFunc) *felt.Felt {
	switch {
	case pn.Binary != nil:
		return hash(pn.Binary.LeftHash, pn.Binary.RightHash)
	case pn.Edge != nil:
		length := make([]byte, len(pn.Edge.Path.bitset))
		length[len(pn.Edge.Path.bitset)-1] = pn.Edge.Path.len
		pathFelt := pn.Edge.Path.Felt()
		lengthFelt := new(felt.Felt).SetBytes(length)
		return new(felt.Felt).Add(hash(pn.Edge.Child, &pathFelt), lengthFelt)
	default:
		return nil
	}
}

func (pn *ProofNode) Len() uint8 {
	if pn.Binary != nil {
		return 1
	}
	return pn.Edge.Path.len
}

func (pn *ProofNode) PrettyPrint() {
	if pn.Binary != nil {
		fmt.Printf("  Binary:\n")
		fmt.Printf("    LeftHash: %v\n", pn.Binary.LeftHash)
		fmt.Printf("    RightHash: %v\n", pn.Binary.RightHash)
	}
	if pn.Edge != nil {
		fmt.Printf("  Edge:\n")
		fmt.Printf("    Child: %v\n", pn.Edge.Child)
		fmt.Printf("    Path: %v\n", pn.Edge.Path)
		fmt.Printf("    Value: %v\n", pn.Edge.Value)
	}
}

type Binary struct {
	LeftHash  *felt.Felt
	RightHash *felt.Felt
}

type Edge struct {
	Child *felt.Felt // child hash
	Path  *Key       // path from parent to child
	Value *felt.Felt // this nodes hash
}

func GetBoundaryProofs(leftBoundary, rightBoundary *Key, tri *Trie) ([2][]ProofNode, error) {
	proofs := [2][]ProofNode{}
	leftProof, err := GetProof(leftBoundary, tri)
	if err != nil {
		return proofs, err
	}
	rightProof, err := GetProof(rightBoundary, tri)
	if err != nil {
		return proofs, err
	}
	proofs[0] = leftProof
	proofs[1] = rightProof
	return proofs, nil
}

func isEdge(parentKey *Key, sNode StorageNode) bool {
	sNodeLen := sNode.key.len
	if parentKey == nil { // Root
		return sNodeLen != 0
	}
	return sNodeLen-parentKey.len > 1
}

// Note: we need to account for the fact that Junos Trie has nodes that are Binary AND Edge,
// whereas the protocol requires nodes that are Binary XOR Edge
func transformNode(tri *Trie, parentKey *Key, sNode StorageNode) (*Edge, *Binary, error) {
	isEdgeBool := isEdge(parentKey, sNode)

	var edge *Edge
	if isEdgeBool {
		edgePath := path(sNode.key, parentKey)
		edge = &Edge{
			Path:  &edgePath,
			Child: sNode.node.Value,
		}
	}
	if sNode.key.len == tri.height { // Leaf
		return edge, nil, nil
	}
	lNode, err := tri.GetNodeFromKey(sNode.node.Left)
	if err != nil {
		return nil, nil, err
	}
	rNode, err := tri.GetNodeFromKey(sNode.node.Right)
	if err != nil {
		return nil, nil, err
	}

	rightHash := rNode.Value
	if isEdge(sNode.key, StorageNode{node: rNode, key: sNode.node.Right}) {
		edgePath := path(sNode.node.Right, sNode.key)
		rEdge := ProofNode{Edge: &Edge{
			Path:  &edgePath,
			Child: rNode.Value,
		}}
		rightHash = rEdge.Hash(tri.hash)
	}
	leftHash := lNode.Value
	if isEdge(sNode.key, StorageNode{node: lNode, key: sNode.node.Left}) {
		edgePath := path(sNode.node.Left, sNode.key)
		lEdge := ProofNode{Edge: &Edge{
			Path:  &edgePath,
			Child: lNode.Value,
		}}
		leftHash = lEdge.Hash(tri.hash)
	}
	binary := &Binary{
		LeftHash:  leftHash,
		RightHash: rightHash,
	}

	return edge, binary, nil
}

// https://github.com/eqlabs/pathfinder/blob/main/crates/merkle-tree/src/tree.rs#L514
func GetProof(key *Key, tri *Trie) ([]ProofNode, error) {
	nodesFromRoot, err := tri.nodesFromRoot(key)
	if err != nil {
		return nil, err
	}
	proofNodes := []ProofNode{}

	var parentKey *Key

	for i, sNode := range nodesFromRoot {
		sNodeEdge, sNodeBinary, err := transformNode(tri, parentKey, sNode)
		if err != nil {
			return nil, err
		}
		isLeaf := sNode.key.len == tri.height

		if sNodeEdge != nil && !isLeaf { // Internal Edge
			proofNodes = append(proofNodes, []ProofNode{{Edge: sNodeEdge}, {Binary: sNodeBinary}}...)
		} else if sNodeEdge == nil && !isLeaf { // Internal Binary
			proofNodes = append(proofNodes, []ProofNode{{Binary: sNodeBinary}}...)
		} else if sNodeEdge != nil && isLeaf { // Leaf Edge
			proofNodes = append(proofNodes, []ProofNode{{Edge: sNodeEdge}}...)
		} else if sNodeEdge == nil && sNodeBinary == nil { // sNode is a binary leaf
			break
		}
		parentKey = nodesFromRoot[i].key
	}
	return proofNodes, nil
}

// verifyProof checks if `leafPath` leads from `root` to `leafHash` along the `proofNodes`
// https://github.com/eqlabs/pathfinder/blob/main/crates/merkle-tree/src/tree.rs#L2006
func VerifyProof(root *felt.Felt, key *Key, value *felt.Felt, proofs []ProofNode, hash hashFunc) bool {
	expectedHash := root
	remainingPath := key
	for _, proofNode := range proofs {
		if !proofNode.Hash(hash).Equal(expectedHash) {
			return false
		}
		switch {
		case proofNode.Binary != nil:
			if remainingPath.Test(remainingPath.Len() - 1) {
				expectedHash = proofNode.Binary.RightHash
			} else {
				expectedHash = proofNode.Binary.LeftHash
			}
			remainingPath.RemoveLastBit()
		case proofNode.Edge != nil:
			subKey, err := remainingPath.SubKey(proofNode.Edge.Path.Len())
			if err != nil {
				return false
			}
			if !proofNode.Edge.Path.Equal(subKey) {
				return false
			}
			expectedHash = proofNode.Edge.Child
			remainingPath.Truncate(251 - proofNode.Edge.Path.Len()) //nolint:gomnd
		}
	}

	return expectedHash.Equal(value)
}

// VerifyRangeProof verifies the range proof for the given range of keys.
// This is achieved by constructing a trie from the boundary proofs, and the supplied key-values.
// If the root of the reconstructed trie matches the supplied root, then the verification passes.
// If the trie is constructed incorrectly then the root will have an incorrect key(len,path), and value,
// and therefore it's hash won't match the expected root
// ref: https://github.com/ethereum/go-ethereum/blob/v1.14.3/trie/proof.go#L484
func VerifyRangeProof(root *felt.Felt, keys, values []*felt.Felt, proofKeys [2]*Key, proofValues [2]*felt.Felt,
	proofs [2][]ProofNode, hash hashFunc,
) (bool, error) {
	// Step 0: checks
	if len(keys) != len(values) {
		return false, fmt.Errorf("inconsistent proof data, number of keys: %d, number of values: %d", len(keys), len(values))
	}

	// Ensure all keys are monotonic increasing
	if err := ensureMonotonicIncreasing(proofKeys, keys); err != nil {
		return false, err
	}

	// Ensure the inner values contain no deletions
	for _, value := range values {
		if value.Equal(&felt.Zero) {
			return false, errors.New("range contains deletion")
		}
	}

	// Step 1: Verify proofs, and get proof paths
	var proofPaths [2][]StorageNode
	var err error
	for i := 0; i < 2; i++ {
		if proofs[i] != nil {
			if !VerifyProof(root, proofKeys[i], proofValues[i], proofs[i], hash) {
				return false, fmt.Errorf("invalid proof for key %x", proofKeys[i].String())
			}

			proofPaths[i], err = ProofToPath(proofs[i], proofKeys[i], hash)
			if err != nil {
				return false, err
			}
		}
	}

	// Step 2: Build trie from proofPaths and keys
	tmpTrie, err := BuildTrie(proofPaths[0], proofPaths[1], keys, values)
	if err != nil {
		return false, err
	}

	// Verify that the recomputed root hash matches the provided root hash
	recomputedRoot, err := tmpTrie.Root()
	if err != nil {
		return false, err
	}
	if !recomputedRoot.Equal(root) {
		return false, errors.New("root hash mismatch")
	}

	return true, nil
}

func ensureMonotonicIncreasing(proofKeys [2]*Key, keys []*felt.Felt) error {
	if proofKeys[0] != nil {
		leftProofFelt := proofKeys[0].Felt()
		if leftProofFelt.Cmp(keys[0]) >= 0 {
			return errors.New("range is not monotonically increasing")
		}
	}
	if proofKeys[1] != nil {
		rightProofFelt := proofKeys[1].Felt()
		if keys[len(keys)-1].Cmp(&rightProofFelt) >= 0 {
			return errors.New("range is not monotonically increasing")
		}
	}
	if len(keys) >= 2 {
		for i := 0; i < len(keys)-1; i++ {
			if keys[i].Cmp(keys[i+1]) >= 0 {
				return errors.New("range is not monotonically increasing")
			}
		}
	}
	return nil
}

// shouldSquish determines if the node needs compressed, and if so, the len needed to arrive at the next key
func shouldSquish(idx int, proofNodes []ProofNode) (int, uint8) {
	parent := &proofNodes[idx]
	var child *ProofNode
	// The child is nil of the current node is a leaf
	if idx != len(proofNodes)-1 {
		child = &proofNodes[idx+1]
	}

	if child == nil {
		return 0, 0
	}

	if parent.Edge != nil && child.Binary != nil {
		return 1, parent.Edge.Path.len
	}

	if parent.Binary != nil && child.Edge != nil {
		return 1, child.Edge.Path.len
	}

	return 0, 0
}

func assignChild(crntNode *Node, nilKey, childKey *Key, isRight bool) {
	if isRight {
		crntNode.Right = childKey
		crntNode.Left = nilKey
	} else {
		crntNode.Right = nilKey
		crntNode.Left = childKey
	}
}

// ProofToPath returns the set of storage nodes along the proofNodes towards the leaf.
// Note that only the nodes and children along this path will be set correctly.
func ProofToPath(proofNodes []ProofNode, leafKey *Key, hashF hashFunc) ([]StorageNode, error) {
	pathNodes := []StorageNode{}

	// Hack: this allows us to store a right without an existing left node.
	zeroFeltBytes := new(felt.Felt).Bytes()
	nilKey := NewKey(0, zeroFeltBytes[:])

	i, offset := 0, 0
	for i <= len(proofNodes)-1 {
		var crntKey *Key
		crntNode := Node{}

		height := getHeight(i, pathNodes, proofNodes)

		// Set the key of the current node
		var err error
		squishedParent, squishParentOffset := shouldSquish(i, proofNodes)
		if proofNodes[i].Binary != nil {
			crntKey, err = leafKey.SubKey(height)
		} else {
			crntKey, err = leafKey.SubKey(height + squishParentOffset)
		}
		if err != nil {
			return nil, err
		}
		offset += squishedParent

		// Set the value of the current node
		crntNode.Value = proofNodes[i].Hash(hashF)

		// Set the children of the current node
		childIdx := i + squishedParent + 1
		childIsRight := leafKey.Test(leafKey.len - crntKey.len - 1)
		if i+2+squishedParent < len(proofNodes)-1 { // The child will be compressed, so point to its compressed form
			squishedChild, squishChildOffset := shouldSquish(childIdx, proofNodes)
			childKey, err := leafKey.SubKey(height + squishParentOffset + squishChildOffset + uint8(squishedChild))
			if err != nil {
				return nil, err
			}
			assignChild(&crntNode, &nilKey, childKey, childIsRight)
		} else if i+1+offset == len(proofNodes)-1 { // The child points to a leaf, keep it as is
			if proofNodes[childIdx].Edge != nil {
				assignChild(&crntNode, &nilKey, leafKey, childIsRight)
			} else {
				childKey, err := leafKey.SubKey(crntKey.len + proofNodes[childIdx].Len())
				if err != nil {
					return nil, err
				}
				assignChild(&crntNode, &nilKey, childKey, childIsRight)
			}
		} else { // Current node points directly to leaf
			if proofNodes[i].Edge != nil && len(pathNodes) > 0 {
				break
			}
			assignChild(&crntNode, &nilKey, leafKey, childIsRight)
		}

		pathNodes = append(pathNodes, StorageNode{key: crntKey, node: &crntNode})
		i += 1 + offset
	}
	pathNodes = addLeafNode(proofNodes, pathNodes, leafKey)
	return pathNodes, nil
}

// getHeight returns the height of the current node, which depends on the previous
// height and whether the current proofnode is edge or binary
func getHeight(idx int, pathNodes []StorageNode, proofNodes []ProofNode) uint8 {
	if len(pathNodes) > 0 {
		if proofNodes[idx].Edge != nil {
			return pathNodes[len(pathNodes)-1].key.len + proofNodes[idx].Edge.Path.len
		} else {
			return pathNodes[len(pathNodes)-1].key.len + 1
		}
	} else {
		return 0
	}
}

// addLeafNode appends the leaf node, if the final node in pathNodes points to a leaf.
func addLeafNode(proofNodes []ProofNode, pathNodes []StorageNode, leafKey *Key) []StorageNode {
	lastNode := pathNodes[len(pathNodes)-1].node
	lastProof := proofNodes[len(proofNodes)-1]
	if lastNode.Left.Equal(leafKey) || lastNode.Right.Equal(leafKey) {
		leafNode := Node{}
		if lastProof.Edge != nil {
			leafNode.Value = lastProof.Edge.Child
		} else if lastNode.Left.Equal(leafKey) {
			leafNode.Value = lastProof.Binary.LeftHash
		} else {
			leafNode.Value = lastProof.Binary.RightHash
		}
		pathNodes = append(pathNodes, StorageNode{key: leafKey, node: &leafNode})
	}
	return pathNodes
}

// BuildTrie builds a trie using the proof paths (including inner nodes), and then sets all the keys-values (leaves)
func BuildTrie(leftProofPath, rightProofPath []StorageNode, keys, values []*felt.Felt) (*Trie, error) {
	tempTrie, err := NewTriePedersen(newMemStorage(), 251) //nolint:gomnd
	if err != nil {
		return nil, err
	}

	// merge proof paths
	for i := range min(len(leftProofPath), len(rightProofPath)) {
		if leftProofPath[i].key.Equal(rightProofPath[i].key) {
			leftProofPath[i].node.Right = rightProofPath[i].node.Right
			rightProofPath[i].node.Left = leftProofPath[i].node.Left
		} else {
			break
		}
	}

	for _, sNode := range leftProofPath {
		_, err := tempTrie.PutInner(sNode.key, sNode.node)
		if err != nil {
			return nil, err
		}
	}

	for _, sNode := range rightProofPath {
		_, err := tempTrie.PutInner(sNode.key, sNode.node)
		if err != nil {
			return nil, err
		}
	}

	for i := range len(keys) {
		_, err := tempTrie.PutWithProof(keys[i], values[i], leftProofPath, rightProofPath)
		if err != nil {
			return nil, err
		}
	}
	return tempTrie, nil
}

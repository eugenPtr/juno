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

func PrettyPrintProofPath(proofPath []ProofNode, key *felt.Felt) {
	fmt.Println("---------------- proof to ", key.String(), " ------------")
	for _, node := range proofPath {
		node.PrettyPrint()
	}
}

type Binary struct {
	LeftHash  *felt.Felt
	RightHash *felt.Felt
}

type Edge struct {
	Child *felt.Felt // Child hash
	Path  *Key       // path from parent to child
	Value *felt.Felt // this nodes hash
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
func adaptNodeToSnap(tri *Trie, parentKey *Key, sNode StorageNode) (*Edge, *Binary, error) {
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
func GetProof(leaf *felt.Felt, tri *Trie) ([]ProofNode, error) {
	leafKey := tri.feltToKey(leaf)
	nodesToLeaf, err := tri.nodesFromRoot(&leafKey)
	if err != nil {
		return nil, err
	}
	proofNodes := []ProofNode{}

	var parentKey *Key

	for i := 0; i < len(nodesToLeaf); i++ {
		sNode := nodesToLeaf[i]
		sNodeEdge, sNodeBinary, err := adaptNodeToSnap(tri, parentKey, sNode)
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
		parentKey = nodesToLeaf[i].key
	}
	return proofNodes, nil
}

func GetBoundaryProofs(leftBoundary, rightBoundary *felt.Felt, tri *Trie) ([2][]ProofNode, error) {
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

// verifyProof checks if `leafPath` leads from `root` to `leafHash` along the `proofNodes`
// https://github.com/eqlabs/pathfinder/blob/main/crates/merkle-tree/src/tree.rs#L2006
func VerifyProof(root *felt.Felt, keyFelt *felt.Felt, value *felt.Felt, proofs []ProofNode, hash hashFunc) bool {
	keyBytes := keyFelt.Bytes()
	key := NewKey(251, keyBytes[:])

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
			if !proofNode.Edge.Path.Equal(remainingPath.SubKey(proofNode.Edge.Path.Len())) {
				return false
			}
			expectedHash = proofNode.Edge.Child
			remainingPath.Truncate(proofNode.Edge.Path.Len())
		}
	}

	return expectedHash.Equal(value)
}

// VerifyRangeProof verifies the range proof for the given range of keys.
// This is achieved by constructing a trie from the boundary proofs, and the supplied key-values.
// If the root of the reconstructed trie matches the supplied root, then the verification passes.
// If the trie is constructed incorrectly then the root will have an incorrect key(len,path), and value,
// and therefore it's hash will be incorrect.
// ref: https://github.com/ethereum/go-ethereum/blob/v1.14.3/trie/proof.go#L484
// Note: this currently assumes that the inner keys do not contain the min/max key (ie both proofs exist) // Todo
// The first/last key and value must correspond to the left/right proofs //Todo we currently assume both proofs are provided, as above
func VerifyRangeProof(root *felt.Felt, keys []*felt.Felt, values []*felt.Felt, proofs [2][]ProofNode, proofKeys [2]*felt.Felt, hash hashFunc) (bool, error) {
	// Step 0: checks
	if len(keys) != len(values) {
		return false, fmt.Errorf("inconsistent proof data, keys: %d, values: %d", len(keys), len(values))
	}
	// Ensure all keys are monotonic increasing
	for i := range keys[0 : len(keys)-2] {
		if keys[i].Cmp(keys[i+1]) >= 0 {
			return false, errors.New("range is not monotonically increasing")
		}
	}
	// Ensure the inner values contain no deletions
	for _, value := range values[1 : len(values)-2] {
		if value.Equal(&felt.Zero) {
			return false, errors.New("range contains deletion")
		}
	}

	// Step 1: Verify the two boundary proofs
	if !VerifyProof(root, keys[0], values[0], proofs[0], hash) {
		return false, fmt.Errorf("invalid proof for key %x", keys[0])
	}
	if !VerifyProof(root, keys[len(keys)-1], values[len(values)-1], proofs[1], hash) {
		return false, fmt.Errorf("invalid proof for key %x", keys[len(keys)-1])
	}

	// Step 2: Get proof paths
	firstProofPath, err := ProofToPath(proofs[0], proofKeys[0], hash)
	if err != nil {
		return false, err
	}

	lastProofPath, err := ProofToPath(proofs[1], proofKeys[1], hash)
	if err != nil {
		return false, err
	}

	// Step 3: Build trie from proofPaths and keys
	tmpTrie, err := BuildTrie(firstProofPath, lastProofPath, keys, values)
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

// Only the path down to leaf Key will be set correctly. Not neightbouring keys
func ProofToPath(proofNodes []ProofNode, leaf *felt.Felt, hashF hashFunc) ([]StorageNode, error) {

	shouldSquish := func(parent *ProofNode, child *ProofNode) (uint8, uint8) {
		if parent == nil || child == nil {
			return 0, 0
		}
		if parent.Edge != nil {
			if child.Binary != nil {
				return 1, uint8(parent.Edge.Path.len)
			}
			return 0, 0
		}
		if parent.Binary != nil {
			if child.Edge != nil {
				return 1, uint8(child.Edge.Path.len)
			}
			return 0, 0
		}

		return 0, 0
	}

	getLen := func(pNode *ProofNode) uint8 {
		if pNode.Binary != nil {
			return 1
		}
		return pNode.Edge.Path.len
	}

	leafBytes := leaf.Bytes()
	leafKey := NewKey(251, leafBytes[:])
	height := uint8(0)
	pathNodes := []StorageNode{}

	i := 0
	offset := 0
	// totalOffset := 0
	for i <= len(proofNodes)-1 {
		var crntKey *Key
		crntNode := Node{}

		nxtProofNode := ProofNode{}
		// Last node (should be binary, last edge accounted for elsewhere)
		if i != len(proofNodes)-1 {
			nxtProofNode = proofNodes[i+1]
		}
		// Set Key of current node
		squishedParent, squishParentOffset := shouldSquish(&proofNodes[i], &nxtProofNode)
		if proofNodes[i].Binary != nil {
			crntKey = leafKey.SubKey(height)
		} else {
			crntKey = leafKey.SubKey(height + squishParentOffset)
		}
		offset += int(squishedParent)

		// Set value
		crntNode.Value = proofNodes[i].Hash(hashF)

		// Check if there is a left/right node that we need to account for as well
		if i+2+int(squishedParent) < len(proofNodes)-1 {
			squishedChild, squishChildOffset := shouldSquish(&proofNodes[i+1+int(squishedParent)], &proofNodes[i+2+int(squishedParent)])
			childKey := leafKey.SubKey(height + squishParentOffset + squishChildOffset + squishedChild)
			if leafKey.Test(leafKey.len - crntKey.len - 1) {
				crntNode.Right = childKey
			} else {
				crntNode.Left = childKey
			}
		} else if i+1+offset == len(proofNodes)-1 { // i+1+offset is the final node
			// If the (squished) child is an edge, then squish
			if proofNodes[i+int(squishedParent)+1].Edge != nil {
				if leafKey.Test(leafKey.len - crntKey.len - 1) {
					crntNode.Right = &leafKey
				} else {
					crntNode.Left = &leafKey
				}
				offset++ // Hack
			} else {
				qwe := getLen(&proofNodes[i+int(squishedParent)+1])
				if leafKey.Test(leafKey.len - crntKey.len - 1) {
					crntNode.Right = leafKey.SubKey(crntKey.len + qwe)
				} else {
					crntNode.Left = leafKey.SubKey(crntKey.len + qwe)
				}
			}
		} else { // Point to leaf
			if leafKey.Test(leafKey.len - crntKey.len - 1) {
				crntNode.Right = &leafKey
			} else {
				crntNode.Left = &leafKey
			}
		}
		height = crntKey.len + 1
		pathNodes = append(pathNodes, StorageNode{key: crntKey, node: &crntNode})
		i += 1 + offset
	}
	return pathNodes, nil
}

// BuildTrie builds a trie using the proof paths (including inner nodes), and then sets all the keys-values (leaves)
// Todo: test
func BuildTrie(firstProofPath, lastProofPath []StorageNode, keys []*felt.Felt, values []*felt.Felt) (*Trie, error) {
	tempTrie, err := NewTriePedersen(newMemStorage(), 251)
	if err != nil {
		return nil, err
	}
	for _, sNode := range firstProofPath {
		err := tempTrie.storage.Put(sNode.key, sNode.node)
		if err != nil {
			return nil, err
		}
	}
	for _, sNode := range lastProofPath {
		err := tempTrie.storage.Put(sNode.key, sNode.node)
		if err != nil {
			return nil, err
		}
	}
	for i := range len(keys) {
		_, err := tempTrie.Put(keys[i], values[i])
		if err != nil {
			return nil, err
		}
	}
	return tempTrie, nil
}

// getExpectedhash effectievly gets the value corresponding to the key given proofs
// https://github.com/eqlabs/pathfinder/blob/main/crates/merkle-tree/src/tree.rs#L2006
// func getExpectedProofValue(root *felt.Felt, proofKey *Key, nbrKey *Key, proofs []ProofNode, hash hashFunc) (*felt.Felt, error) { // Todo: test
// 	if proofKey.Len() != 251 || nbrKey.Len() != 251 { //nolint:gomnd
// 		return nil, errors.New("keys not the correct length")
// 	}

// 	commonAncestorKey := nbrKey.commonPrefix(*proofKey)
// 	expectedHash := root
// 	remainingPath := proofKey
// 	height := uint8(0)
// 	for _, proofNode := range proofs {
// 		if !proofNode.Hash(hash).Equal(expectedHash) {
// 			return nil, errors.New("proofNode not the expected hash")
// 		}
// 		if height == commonAncestorKey.len {
// 			return proofNode.Hash(hash), nil
// 		}
// 		switch {
// 		case proofNode.Binary != nil:
// 			if remainingPath.Test(remainingPath.Len() - 1) {
// 				expectedHash = proofNode.Binary.RightHash
// 			} else {
// 				expectedHash = proofNode.Binary.LeftHash
// 			}
// 			remainingPath.RemoveLastBit()
// 			height++
// 		case proofNode.Edge != nil:
// 			if !proofNode.Edge.Path.Equal(remainingPath.SubKey(proofNode.Edge.Path.Len())) {
// 				return nil, errors.New("error in key path")
// 			}
// 			expectedHash = proofNode.Edge.Child
// 			remainingPath.Truncate(proofNode.Edge.Path.Len())
// 			height += proofNode.Edge.Path.Len()
// 		}

// 	}
// 	return nil, errors.New("failed to get proof value")
// }

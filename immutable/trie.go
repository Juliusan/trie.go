package immutable

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/trie.go/common"
)

// Trie is an updatable trie implemented on top of the unpackedKey/value store. It is virtualized and optimized by caching of the
// trie update operation and keeping consistent trie in the cache
type Trie struct {
	TrieReader
	mutatedRoot *bufferedNode
}

// TrieReader direct read-only access to trie
type TrieReader struct {
	nodeStore      *NodeStore
	persistentRoot common.VCommitment
}

func NewTrie(nodeStore *NodeStore, root common.VCommitment) (*Trie, error) {
	rootNodeData, ok := nodeStore.FetchNodeData(root)
	if !ok {
		return nil, fmt.Errorf("root commitment '%s' does not exist", root)
	}
	ret := &Trie{
		TrieReader: TrieReader{
			persistentRoot: root,
			nodeStore:      nodeStore,
		},
		mutatedRoot: newBufferedNode(rootNodeData, nil),
	}
	return ret, nil
}

func (tr *TrieReader) RootCommitment() common.VCommitment {
	return tr.persistentRoot
}

func (tr *TrieReader) Model() common.CommitmentModel {
	return tr.nodeStore.m
}

func (tr *TrieReader) PathArity() common.PathArity {
	return tr.nodeStore.m.PathArity()
}

// Commit calculates a new mutatedRoot commitment value from the cache and commits all mutations in the cached TrieReader
// It is a re-calculation of the trie. bufferedNode caches are updated accordingly.
func (tr *Trie) Commit(w common.KVWriter) common.VCommitment {
	commitNode(w, tr.Model(), tr.mutatedRoot)
	return tr.mutatedRoot.nodeData.Commitment
}

// commitNode re-calculates node commitment and, recursively, its children commitments
// Child modification marks in 'uncommittedChildren' are updated
// Return update to the upper commitment. nil mean upper commitment is not updated
// It calls implementation-specific function UpdateNodeCommitment and passes parameter
// calcDelta = true if node's commitment can be updated incrementally. The implementation
// of UpdateNodeCommitment may use this parameter to optimize underlying cryptography
//
// commitNode does not commit to the state index
func commitNode(w common.KVWriter, m common.CommitmentModel, node *bufferedNode) {
	childUpdates := make(map[byte]common.VCommitment)
	for idx, child := range node.uncommittedChildren {
		if child == nil {
			childUpdates[idx] = nil
		} else {
			commitNode(w, m, child)
			childUpdates[idx] = child.nodeData.Commitment
		}
	}
	m.UpdateNodeCommitment(node.nodeData, childUpdates, node.terminal, node.pathFragment, !common.IsNil(node.nodeData.Commitment))
	node.uncommittedChildren = make(map[byte]*bufferedNode)
}

// Update updates Trie with the unpackedKey/value. Reorganizes and re-calculates trie, keeps cache consistent
func (tr *Trie) Update(triePath []byte, value []byte) {
	unpackedTriePath := common.UnpackBytes(triePath, tr.PathArity())
	if len(value) == 0 {
		tr.delete(unpackedTriePath)
	} else {
		tr.update(unpackedTriePath, value)
	}
}

// Delete deletes Key/value from the Trie, reorganizes the trie
func (tr *Trie) Delete(key []byte) {
	tr.Update(key, nil)
}

// PersistMutations persists/append the cache to the store.
// Returns deleted part for possible use in the mutable state implementation
// Does not clear cache
func (tr *Trie) PersistMutations(store common.KVWriter) (int, map[string]struct{}) {
	panic("implement me")
}

// UpdateStr updates unpackedKey/value pair in the trie
func (tr *Trie) UpdateStr(key interface{}, value interface{}) {
	var k, v []byte
	if key != nil {
		switch kt := key.(type) {
		case []byte:
			k = kt
		case string:
			k = []byte(kt)
		default:
			panic("[]byte or string expected")
		}
	}
	if value != nil {
		switch vt := value.(type) {
		case []byte:
			v = vt
		case string:
			v = []byte(vt)
		default:
			panic("[]byte or string expected")
		}
	}
	tr.Update(k, v)
}

// DeleteStr removes node from trie
func (tr *Trie) DeleteStr(key interface{}) {
	var k []byte
	if key != nil {
		switch kt := key.(type) {
		case []byte:
			k = kt
		case string:
			k = []byte(kt)
		default:
			panic("[]byte or string expected")
		}
	}
	tr.Delete(k)
}

func (tr *Trie) newTerminalNode(triePath, pathFragment, value []byte) *bufferedNode {
	ret := newBufferedNode(nil, triePath)
	ret.setPathFragment(pathFragment)
	ret.setValue(value, tr.Model())
	return ret
}

func (tr *Trie) VectorCommitmentFromBytes(data []byte) (common.VCommitment, error) {
	ret := tr.nodeStore.m.NewVectorCommitment()
	rdr := bytes.NewReader(data)
	if err := ret.Read(rdr); err != nil {
		return nil, err
	}
	if rdr.Len() != 0 {
		return nil, common.ErrNotAllBytesConsumed
	}
	return ret, nil
}

// Reconcile returns a list of keys in the store which cannot be proven in the trie
// Trie is consistent if empty slice is returned
// May be an expensive operation
func (tr *Trie) Reconcile(store common.KVIterator) [][]byte {
	panic("implement me")
	//ret := make([][]byte, 0)
	//store.Iterate(func(k, v []byte) bool {
	//	p, _, ending := proofPath(tr, UnpackBytes(k, tr.PathArity()))
	//	if ending == EndingTerminal {
	//		lastKey := p[len(p)-1]
	//		n, ok := tr.GetNode(lastKey)
	//		if !ok {
	//			ret = append(ret, k)
	//		} else {
	//			if !tr.Model().EqualCommitments(tr.trieBuffer.nodeStore.m.CommitToData(v), n.terminal()) {
	//				ret = append(ret, k)
	//			}
	//		}
	//	} else {
	//		ret = append(ret, k)
	//	}
	//	return true
	//})
	//return ret
}

// UpdateAll mass-updates trie from the unpackedKey/value store.
// To be used to build trie for arbitrary unpackedKey/value data sets
func (tr *Trie) UpdateAll(store common.KVIterator) {
	store.Iterate(func(k, v []byte) bool {
		tr.Update(k, v)
		return true
	})
}

func (tr *Trie) DangerouslyDumpCacheToString() string {
	panic("implement me")
	//return tr.trieBuffer.dangerouslyDumpCacheToString()
}

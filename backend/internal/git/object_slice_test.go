package git_test

import (
	"reflect"
	"testing"

	gitrepo "gogit/internal/git"
)

func TestEncodeTreeDoesNotMutateEntries(t *testing.T) {
	blobOID := gitrepo.SHA1.Sum(gitrepo.EncodeObject(gitrepo.TypeBlob, []byte("payload")))
	treeOID := gitrepo.SHA1.Sum(gitrepo.EncodeObject(gitrepo.TypeTree, nil))
	entries := []gitrepo.TreeEntry{
		{Name: "z.txt", OID: blobOID},
		{Mode: "040000", Name: "assets", OID: treeOID},
		{Mode: "100755", Name: "build", OID: blobOID, Type: gitrepo.TypeBlob},
	}
	want := append([]gitrepo.TreeEntry(nil), entries...)

	if _, err := gitrepo.EncodeTree(entries, gitrepo.SHA1); err != nil {
		t.Fatalf("EncodeTree: %v", err)
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("EncodeTree mutated caller entries:\n got: %#v\nwant: %#v", entries, want)
	}
}

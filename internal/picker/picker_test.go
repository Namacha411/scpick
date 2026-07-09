package picker

import (
	"reflect"
	"testing"
)

func TestBuildFileList(t *testing.T) {
	entries := []Entry{
		{Name: "b.txt", Path: "/dir/b.txt", IsDir: false},
		{Name: "sub2", Path: "/dir/sub2", IsDir: true},
		{Name: "a.txt", Path: "/dir/a.txt", IsDir: false},
		{Name: "sub1", Path: "/dir/sub1", IsDir: true},
	}

	got := BuildFileList(entries, "/parent")

	want := []ListItem{
		{Label: "..", Path: "/parent", IsDir: true},
		{Label: "sub1", Path: "/dir/sub1", IsDir: true},
		{Label: "sub2", Path: "/dir/sub2", IsDir: true},
		{Label: "a.txt", Path: "/dir/a.txt", IsDir: false},
		{Label: "b.txt", Path: "/dir/b.txt", IsDir: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildFileList() = %+v, want %+v", got, want)
	}
}

func TestBuildFileListEmpty(t *testing.T) {
	got := BuildFileList(nil, "/parent")
	want := []ListItem{{Label: "..", Path: "/parent", IsDir: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildFileList(nil) = %+v, want %+v", got, want)
	}
}

func TestBuildDirList(t *testing.T) {
	entries := []Entry{
		{Name: "file.txt", Path: "/dir/file.txt", IsDir: false},
		{Name: "sub2", Path: "/dir/sub2", IsDir: true},
		{Name: "sub1", Path: "/dir/sub1", IsDir: true},
	}

	got := BuildDirList(entries, "/dir", "/parent")

	want := []ListItem{
		{Label: "★ use this dir", Path: "/dir", IsMarker: true},
		{Label: "..", Path: "/parent", IsDir: true},
		{Label: "sub1", Path: "/dir/sub1", IsDir: true},
		{Label: "sub2", Path: "/dir/sub2", IsDir: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildDirList() = %+v, want %+v", got, want)
	}
	for _, item := range got {
		if !item.IsMarker && item.Label == "file.txt" {
			t.Errorf("BuildDirList must not include files, got %+v", got)
		}
	}
}

func TestBuildDirListMarkerAlwaysFirst(t *testing.T) {
	got := BuildDirList(nil, "/dir", "/parent")
	if len(got) < 2 || !got[0].IsMarker || got[1].Label != ".." {
		t.Errorf("expected marker then .. as first two items, got %+v", got)
	}
}

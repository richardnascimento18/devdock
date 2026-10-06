package core

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestMoveUnsupportedNoReplace(t *testing.T) {
	for _, errno := range []syscall.Errno{syscall.EINVAL, syscall.ENOSYS, syscall.EOPNOTSUPP, syscall.ENOTSUP} {
		t.Run(errno.Error(), func(t *testing.T) {
			root := t.TempDir()
			p, err := CreateProject(Location{Root: root, Domain: "from"}, "demo")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(p.Path, "data"), []byte("payload"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("data", filepath.Join(p.Path, "link")); err != nil {
				t.Fatal(err)
			}
			plan, err := PlanMove(p, Location{Root: root, Domain: "to"}, "demo")
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			mover := Mover{NoReplace: func(string, string) error { calls++; return errno }}
			moved, err := mover.Execute(plan)
			if err != nil || calls != 1 || moved.Path != plan.Path {
				t.Fatalf("move: %+v %v (%d)", moved, err, calls)
			}
			if data, err := os.ReadFile(filepath.Join(moved.Path, "data")); err != nil || string(data) != "payload" {
				t.Fatalf("data: %s %v", data, err)
			}
			if link, err := os.Readlink(filepath.Join(moved.Path, "link")); err != nil || link != "data" {
				t.Fatalf("link: %s %v", link, err)
			}
			if _, err := os.Lstat(p.Path); !os.IsNotExist(err) {
				t.Fatal("source retained after success")
			}
		})
	}
}

func TestPortableRenameFilesAndSymlinks(t *testing.T) {
	for _, kind := range []string{"file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			src, dst := filepath.Join(t.TempDir(), "source"), filepath.Join(t.TempDir(), "destination")
			if kind == "file" {
				if err := os.WriteFile(src, []byte("data"), 0600); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink("missing", src); err != nil {
				t.Fatal(err)
			}
			mover := Mover{NoReplace: func(string, string) error { return syscall.EINVAL }}
			if err := mover.rename(src, dst); err != nil {
				t.Fatal(err)
			}
			if kind == "symlink" {
				if link, err := os.Readlink(dst); err != nil || link != "missing" {
					t.Fatalf("symlink: %s %v", link, err)
				}
			} else if data, err := os.ReadFile(dst); err != nil || string(data) != "data" {
				t.Fatalf("file: %s %v", data, err)
			}
		})
	}
}

func TestMoveCopyPublicationCompatibility(t *testing.T) {
	for _, collision := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "collision"}[collision], func(t *testing.T) {
			root := t.TempDir()
			p, err := CreateProject(Location{Root: root, Domain: "from"}, "demo")
			if err != nil {
				t.Fatal(err)
			}
			plan, err := PlanMove(p, Location{Root: root, Domain: "to"}, "demo")
			if err != nil {
				t.Fatal(err)
			}
			mover := Mover{NoReplace: func(src, dst string) error {
				if src == p.Path {
					return syscall.EXDEV
				}
				if collision {
					if err := os.Mkdir(dst, 0755); err != nil {
						return err
					}
				}
				return syscall.EINVAL
			}}
			_, err = mover.Execute(plan)
			if collision {
				if !errors.Is(err, os.ErrExist) {
					t.Fatalf("collision: %v", err)
				}
				if _, err := os.Lstat(p.Path); err != nil {
					t.Fatalf("lost source: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			stages, err := filepath.Glob(filepath.Join(root, "to", ".devdock-move-*"))
			if err != nil || len(stages) != 0 {
				t.Fatalf("stage leak: %v %v", stages, err)
			}
		})
	}
}

func TestMoveCompatibilityFailureRetainsSource(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "publication denied", true: "late destination symlink"}[existing], func(t *testing.T) {
			root := t.TempDir()
			p, err := CreateProject(Location{Root: root, Domain: "from"}, "demo")
			if err != nil {
				t.Fatal(err)
			}
			plan, err := PlanMove(p, Location{Root: root, Domain: "to"}, "demo")
			if err != nil {
				t.Fatal(err)
			}
			mover := Mover{NoReplace: func(src, dst string) error {
				if existing {
					if err := os.Symlink("missing", dst); err != nil {
						return err
					}
				}
				return syscall.ENOSYS
			}, Rename: func(string, string) error { return syscall.EACCES }}
			_, err = mover.Execute(plan)
			want := error(syscall.EACCES)
			if existing {
				want = os.ErrExist
			}
			if !errors.Is(err, want) {
				t.Fatalf("want %v: %v", want, err)
			}
			if _, err := os.Lstat(p.Path); err != nil {
				t.Fatalf("source lost: %v", err)
			}
		})
	}
}

func TestNoReplaceSuccessAndCollision(t *testing.T) {
	root := t.TempDir()
	src, dst := filepath.Join(root, "src"), filepath.Join(root, "dst")
	if err := os.Mkdir(src, 0755); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := renameCompatible(src, dst, renameExclusive, func(string, string) error { called = true; return nil }); err != nil || called {
		t.Fatalf("fast path: %v %t", err, called)
	}
	if err := os.Mkdir(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := (Mover{}).rename(src, dst); !errors.Is(err, os.ErrExist) {
		t.Fatalf("collision: %v", err)
	}
}

package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseURL(t *testing.T) {
	cases := map[string]string{
		"":       "https://api.github.com/repos/voocel/ainovel-cli/releases/latest",
		"latest": "https://api.github.com/repos/voocel/ainovel-cli/releases/latest",
		"1.2.3":  "https://api.github.com/repos/voocel/ainovel-cli/releases/tags/v1.2.3",
		"v1.2.3": "https://api.github.com/repos/voocel/ainovel-cli/releases/tags/v1.2.3",
	}
	for target, want := range cases {
		if got := releaseURL("voocel/ainovel-cli", target); got != want {
			t.Fatalf("releaseURL(%q) = %q, want %q", target, got, want)
		}
	}
}

func TestSelectAsset(t *testing.T) {
	suffix, err := assetSuffix()
	if err != nil {
		t.Skip(err)
	}
	rel := &release{
		TagName: "v1.2.3",
		Assets: []releaseAsset{
			{Name: "ainovel-cli_v1.2.3_Windows_x86_64.zip", BrowserDownloadURL: "wrong"},
			{Name: "ainovel-cli_v1.2.3" + suffix, BrowserDownloadURL: "right"},
		},
	}
	asset, err := selectAsset(rel, "ainovel-cli")
	if err != nil {
		t.Fatalf("selectAsset: %v", err)
	}
	if asset.BrowserDownloadURL != "right" {
		t.Fatalf("asset = %+v", asset)
	}
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "ainovel-cli")
	src := filepath.Join(dir, "new")
	if err := os.WriteFile(dst, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("new"), 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := replaceExecutable(dst, src)
	if err != nil {
		t.Fatalf("replaceExecutable: %v", err)
	}
	realDst, err := filepath.EvalSymlinks(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got != realDst {
		t.Fatalf("path = %q, want %q", got, realDst)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("content = %q", data)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
	if _, err := os.Stat(dst + ".old"); !os.IsNotExist(err) {
		t.Fatalf("backup should be removed, err=%v", err)
	}
}

func TestSelectChecksumAssetRequired(t *testing.T) {
	rel := &release{TagName: "v1.2.3", Assets: []releaseAsset{{Name: "ainovel-cli_v1.2.3_Linux_x86_64.tar.gz", BrowserDownloadURL: "x"}}}
	if _, err := selectChecksumAsset(rel); err == nil {
		t.Fatal("release thiếu checksums phải bị từ chối")
	}
	rel.Assets = append(rel.Assets, releaseAsset{Name: "ainovel-cli_checksums.txt", BrowserDownloadURL: "sum"})
	asset, err := selectChecksumAsset(rel)
	if err != nil || asset.BrowserDownloadURL != "sum" {
		t.Fatalf("selectChecksumAsset = %+v, %v", asset, err)
	}
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "pkg.tar.gz")
	if err := os.WriteFile(archive, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	// sha256("hello")
	const sum = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	name := "ainovel-cli_1.2.3_Linux_x86_64.tar.gz"
	sums := filepath.Join(dir, "checksums.txt")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(sums, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("deadbeef  other.tar.gz\n" + sum + "  " + name + "\n")
	if err := verifyChecksum(archive, sums, name); err != nil {
		t.Fatalf("checksum đúng phải qua: %v", err)
	}

	write(strings.Repeat("0", 64) + "  " + name + "\n")
	if err := verifyChecksum(archive, sums, name); err == nil {
		t.Fatal("checksum sai phải bị từ chối")
	}

	write(sum + "  other.tar.gz\n")
	if err := verifyChecksum(archive, sums, name); err == nil {
		t.Fatal("thiếu dòng cho asset phải bị từ chối")
	}
}

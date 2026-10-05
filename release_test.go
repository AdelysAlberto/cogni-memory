package cogni_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePublication(test *testing.T) {
	script, err := os.ReadFile("release.sh")
	if err != nil {
		test.Fatal(err)
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		test.Skip("bash is unavailable")
	}
	cases := []struct {
		name          string
		failure       string
		failedCommand string
		buildCount    int
		dmg           bool
		releaseType   string
		expectedTag   string
	}{
		{name: "missing-gh", failure: "missing-gh"},
		{name: "unauthenticated", failure: "auth", failedCommand: "gh auth status"},
		{name: "build-fails", failure: "build", failedCommand: "go build", buildCount: 1},
		{name: "later-build-fails", failure: "build-later", failedCommand: "go build", buildCount: 3},
		{name: "tag-fails", failure: "tag", failedCommand: "git tag"},
		{name: "branch-push-fails", failure: "push-main", failedCommand: "git push origin main"},
		{name: "tag-push-fails", failure: "push-tag", failedCommand: "git push origin v2.3.8"},
		{name: "upload-fails", failure: "create", failedCommand: "gh release create"},
		{name: "publish-fails", failure: "edit", failedCommand: "gh release edit"},
		{name: "success"},
		{name: "success-with-dmg", dmg: true},
		{name: "success-minor", releaseType: "minor", expectedTag: "v2.4.0"},
		{name: "success-major", releaseType: "major", expectedTag: "v3.0.0"},
	}
	for _, testCase := range cases {
		test.Run(testCase.name, func(test *testing.T) {
			workspace := test.TempDir()
			tools := filepath.Join(workspace, "tools")
			if err := os.Mkdir(tools, 0755); err != nil {
				test.Fatal(err)
			}
			for _, name := range []string{"mkdir", "cp", "uname", "tr"} {
				path, err := exec.LookPath(name)
				if err != nil {
					test.Skipf("%s is unavailable", name)
				}
				if err := os.Symlink(path, filepath.Join(tools, name)); err != nil {
					test.Fatal(err)
				}
			}
			writeReleaseFixture(test, filepath.Join(workspace, "release.sh"), string(script))
			writeReleaseFixture(test, filepath.Join(tools, "git"), `#!/bin/sh
printf 'git %s\n' "$*" >> "$RELEASE_TEST_LOG"
case "$1" in
    describe) printf 'v2.3.7\n' ;;
    tag) [ "$RELEASE_TEST_FAILURE" != tag ] || exit 1 ;;
	push)
		if [ "$RELEASE_TEST_FAILURE" = push-main ] && [ "$3" = main ]; then exit 1; fi
		if [ "$RELEASE_TEST_FAILURE" = push-tag ] && [ "$3" = v2.3.8 ]; then exit 1; fi
		;;
esac
`)
			writeReleaseFixture(test, filepath.Join(tools, "go"), `#!/bin/sh
printf 'go %s\n' "$*" >> "$RELEASE_TEST_LOG"
[ "$RELEASE_TEST_FAILURE" != build ] || exit 1
if [ "$RELEASE_TEST_FAILURE" = build-later ] && [ "$GOOS/$GOARCH" = linux/amd64 ]; then exit 1; fi
while [ "$#" -gt 0 ]; do
    if [ "$1" = -o ]; then
        shift
        printf 'test binary\n' > "$1"
        exit 0
    fi
    shift
done
exit 1
`)
			if testCase.failure != "missing-gh" {
				writeReleaseFixture(test, filepath.Join(tools, "gh"), `#!/bin/sh
printf 'gh %s\n' "$*" >> "$RELEASE_TEST_LOG"
if [ "$1" = auth ] && [ "$RELEASE_TEST_FAILURE" = auth ]; then exit 1; fi
if [ "$1" = release ] && [ "$2" = "$RELEASE_TEST_FAILURE" ]; then exit 1; fi
`)
			}
			if err := os.MkdirAll(filepath.Join(workspace, "bin"), 0755); err != nil {
				test.Fatal(err)
			}
			writeReleaseFixture(test, filepath.Join(workspace, "bin", "cogni_stale"), "old binary")
			if testCase.dmg {
				if err := os.MkdirAll(filepath.Join(workspace, "macos", "dist"), 0755); err != nil {
					test.Fatal(err)
				}
				writeReleaseFixture(test, filepath.Join(workspace, "macos", "dist", "CogniBar.dmg"), "test dmg")
			}
			logPath := filepath.Join(workspace, "commands.log")
			releaseType := testCase.releaseType
			if releaseType == "" {
				releaseType = "patch"
			}
			command := exec.Command(bash, "release.sh", releaseType)
			command.Dir = workspace
			command.Env = append(os.Environ(), "PATH="+tools, "RELEASE_TEST_LOG="+logPath, "RELEASE_TEST_FAILURE="+testCase.failure)
			output, runErr := command.CombinedOutput()
			logBytes, readErr := os.ReadFile(logPath)
			if readErr != nil && !os.IsNotExist(readErr) {
				test.Fatal(readErr)
			}
			log := string(logBytes)
			if testCase.failure != "" {
				if runErr == nil || strings.Contains(string(output), "publicado con éxito") {
					test.Fatalf("Expected failure without success announcement: %v\n%s", runErr, output)
				}
				if (testCase.failure == "missing-gh" || testCase.failure == "auth") && strings.Contains(log, "git ") {
					test.Fatalf("Git ran before preflight succeeded: %s", log)
				}
				if testCase.failure != "edit" && strings.Contains(log, "gh release edit") {
					test.Fatalf("Published after an earlier failure: %s", log)
				}
				if testCase.failedCommand != "" {
					commands := strings.Split(strings.TrimSpace(log), "\n")
					if !strings.HasPrefix(commands[len(commands)-1], testCase.failedCommand) {
						test.Fatalf("Expected immediate stop at %s, got: %s", testCase.failedCommand, log)
					}
				}
				if testCase.buildCount > 0 && strings.Count(log, "go build ") != testCase.buildCount {
					test.Fatalf("Expected %d build attempts, got: %s", testCase.buildCount, log)
				}
				return
			}
			if runErr != nil || !strings.Contains(string(output), "publicado con éxito") {
				test.Fatalf("Expected successful publication: %v\n%s", runErr, output)
			}
			expectedTag := testCase.expectedTag
			if expectedTag == "" {
				expectedTag = "v2.3.8"
			}
			if strings.Count(log, "go build ") != 4 || strings.Count(log, "internal/cli.Version="+expectedTag) != 4 {
				test.Fatalf("Expected four binaries built with version %s, got: %s", expectedTag, log)
			}
			if !strings.Contains(log, "git tag -a "+expectedTag) || !strings.Contains(log, "git push origin "+expectedTag) {
				test.Fatalf("Expected tag %s created and pushed, got: %s", expectedTag, log)
			}
			create := "gh release create " + expectedTag + " bin/cogni_darwin_arm64 bin/cogni_darwin_amd64 bin/cogni_linux_amd64 bin/cogni_linux_arm64"
			if testCase.dmg {
				create += " macos/dist/CogniBar.dmg"
			}
			create += " --repo AdelysAlberto/cogni-memory --verify-tag --draft --title " + expectedTag + " --notes Release " + expectedTag
			publish := "gh release edit " + expectedTag + " --repo AdelysAlberto/cogni-memory --draft=false --latest"
			if !strings.Contains(log, create+"\n"+publish) || strings.Contains(log, "cogni_stale") {
				test.Fatalf("Expected exact assets uploaded before publication: %s", log)
			}
			if strings.Index(log, "git tag -a "+expectedTag) >= strings.Index(log, create) || strings.Index(log, "git push origin "+expectedTag) >= strings.Index(log, create) {
				test.Fatalf("Expected tag creation and push before draft release: %s", log)
			}
		})
	}
}

func writeReleaseFixture(test *testing.T, path, content string) {
	test.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		test.Fatal(err)
	}
}

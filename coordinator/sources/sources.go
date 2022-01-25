package sources

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

var ErrGitLogOutOfBounds = errors.New("Requested out-of-bounds git log")

type GitLogRecord struct {
	CommitHash       string       `json:"commit"`
	ParentCommitHash string       `json:"parent"`
	Subject          string       `json:"subject"`
	Author           GitLogPerson `json:"author"`
	AuthoredString   string       `json:"authored_date,omitempty"`
	Authored         time.Time    `json:"authored"`
	Committer        GitLogPerson `json:"committer"`
	CommittedString  string       `json:"committed_date,omitempty"`
	Committed        time.Time    `json:"committed"`
}

type GitLogPerson struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type SourcesManager struct {
	gitLog      []GitLogRecord
	sourcesLock sync.Mutex
}

func NewSourcesManager() *SourcesManager {
	s := &SourcesManager{gitLog: []GitLogRecord{}, sourcesLock: sync.Mutex{}}
	return s
}

func sourcesParentDir() string {
	return common.DataDir()
}

func archivePath(commitHash string) (string, error) {
	archiveDir := filepath.Join(common.DataDir(), "archives")
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		err = os.Mkdir(archiveDir, 0755)
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(archiveDir, fmt.Sprintf("%s.tar.gz", commitHash)), nil
}

func BinariesArchivePath(
	commitHash string,
	profilingOrDebugging bool,
) (string, error) {
	if _, err := os.Stat(binariesDir()); os.IsNotExist(err) {
		err = os.Mkdir(binariesDir(), 0755)
		if err != nil {
			return "", err
		}
	}
	if profilingOrDebugging {
		commitHash = fmt.Sprintf("%s-profiling", commitHash)
	}
	return filepath.Join(
		binariesDir(),
		fmt.Sprintf("%s.tar.gz", commitHash),
	), nil
}

func sourcesDirName() string {
	return "sources"
}

func sourcesDir() string {
	dir := filepath.Join(sourcesParentDir(), sourcesDirName())
	return dir
}

func binariesDir() string {
	dir := filepath.Join(common.DataDir(), "binaries")
	return dir
}

func (s *SourcesManager) EnsureSourcesUpdated() error {
	var err error
	if _, err = os.Stat(sourcesDir()); os.IsNotExist(err) {
		err = s.cloneSources()
	} else {
		err = s.updateSources()
	}
	if err != nil {
		return err
	}
	return s.updateCommitHistory()
}

func (s *SourcesManager) Compile(
	hash string,
	profilingOrDebugging bool,
	progress chan float64,
) error {
	defer func() {
		if progress != nil {
			progress <- 100
			close(progress)
		}
	}()

	binariesPath := filepath.Join(sourcesDir(), "build")
	path, err := BinariesArchivePath(hash, profilingOrDebugging)
	if err != nil {
		return err
	}

	if progress != nil {
		progress <- 1
	}

	s.sourcesLock.Lock()
	defer s.sourcesLock.Unlock()

	if progress != nil {
		progress <- 2
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		// Already exists
		return nil
	}

	cmd := exec.Command("git", "checkout", hash)
	cmd.Dir = sourcesDir()
	err = cmd.Run()
	if err != nil {
		return err
	}
	logging.Infof(
		"[Compile %s-%t]: Checkout complete",
		hash,
		profilingOrDebugging,
	)

	if progress != nil {
		progress <- 5
	}

	cmd = exec.Command("git", "submodule", "update", "--recursive")
	cmd.Dir = sourcesDir()
	err = cmd.Run()
	if err != nil {
		return err
	}
	logging.Infof(
		"[Compile %s-%t]: Update submodules complete",
		hash,
		profilingOrDebugging,
	)

	if progress != nil {
		progress <- 10
	}

	os.RemoveAll(filepath.Join(sourcesDir(), "build"))
	logging.Infof(
		"[Compile %s-%t]: Cleaned build directory",
		hash,
		profilingOrDebugging,
	)

	cmd = exec.Command(
		"bash",
		filepath.Join(sourcesDir(), "scripts", "configure.sh"),
	)
	cmd.Dir = sourcesDir()
	env := os.Environ()
	if !profilingOrDebugging {
		env = append(env, "BUILD_RELEASE=1")
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Configure failed: %v\n\n%v", err, string(out))
	}

	logging.Infof(
		"[Compile %s-%t]: Configure script complete",
		hash,
		profilingOrDebugging,
	)

	if progress != nil {
		progress <- 50
	}

	cmd = exec.Command(
		"bash",
		filepath.Join(sourcesDir(), "scripts", "build.sh"),
	)
	cmd.Dir = sourcesDir()
	env = os.Environ()
	if profilingOrDebugging {
		env = append(env, "BUILD_PROFILING=1")
	} else {
		env = append(env, "BUILD_RELEASE=1")
	}
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Build failed: %v\n\n%v", err, string(out))
	}

	logging.Infof(
		"[Compile %s-%t]: Build script complete",
		hash,
		profilingOrDebugging,
	)
	if progress != nil {
		progress <- 90
	}

	return common.CreateArchive(binariesPath, path)
}

type PRData struct {
	Subject        string `json:"subject"`
	AuthoredString string `json:"authored_date"`
}

func (s *SourcesManager) updateCommitHistory() error {
	s.sourcesLock.Lock()
	defer s.sourcesLock.Unlock()
	cmd := exec.Command(
		"git",
		"log",
		`--pretty=format:{%n  $$$commit$$$: $$$%H$$$,%n  $$$parent$$$: $$$%P$$$,%n  $$$subject$$$: $$$%s$$$, %n  $$$author$$$: {%n    $$$name$$$: $$$%aN$$$,%n    $$$email$$$: $$$%aE$$$ },%n  $$$authored_date$$$: $$$%aD$$$%n ,%n  $$$committer$$$: {%n    $$$name$$$: $$$%cN$$$,%n    $$$email$$$: $$$%cE$$$},%n    $$$committed_date$$$: $$$%cD$$$%n%n},`,
	)
	cmd.Dir = sourcesDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	outString := string(out[:len(out)-1])
	outString = strings.ReplaceAll(outString, "\"", "\\\"")
	outString = strings.ReplaceAll(outString, "$$$", "\"")
	out = []byte(fmt.Sprintf("[%s]", outString))
	newGitLog := []GitLogRecord{}
	err = json.Unmarshal(out, &newGitLog)
	if err != nil {
		return err
	}

	for i := range newGitLog {
		newGitLog[i].Committed, _ = time.Parse(
			"Mon, 2 Jan 2006 15:04:05 -0700",
			newGitLog[i].CommittedString,
		)
		newGitLog[i].Authored, _ = time.Parse(
			"Mon, 2 Jan 2006 15:04:05 -0700",
			newGitLog[i].AuthoredString,
		)
		newGitLog[i].AuthoredString = ""
		newGitLog[i].CommittedString = ""
	}

	s.gitLog = newGitLog[:40]

	cmd = exec.Command(
		"git",
		"fetch",
		"origin",
		"+refs/pull/*/head:refs/remotes/origin/pr/*",
	)
	cmd.Dir = sourcesDir()
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to fetch PRs: %v\n\n%s", err, string(out))
	}

	prs := map[int]string{}

	cmd = exec.Command("git", "show-ref")
	cmd.Dir = sourcesDir()
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to list refs: %v", err)
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.Split(line, " ")
		if len(parts) == 2 {
			if strings.HasPrefix(parts[1], "refs/remotes/origin/pr/") {
				pr, err := strconv.Atoi(parts[1][23:])
				if err == nil {
					prs[pr] = parts[0]
				}
			}
		}
	}

	// Remove PRs that are merged or too old
	for pr, hash := range prs {
		cmd = exec.Command(
			"git",
			"log",
			"-n",
			"1",
			`--pretty=format:{%n  $$$subject$$$: $$$%s$$$, $$$authored_date$$$: $$$%aD$$$%n }`,
			hash,
		)
		cmd.Dir = sourcesDir()
		out, err = cmd.CombinedOutput()
		if err != nil {
			logging.Warnf("git log for PR %d failed: %v", pr, err)
			continue
		}
		outString := strings.ReplaceAll(string(out), "\"", "\\\"")
		outString = strings.ReplaceAll(outString, "$$$", "\"")
		out = []byte(outString)
		var prData PRData
		err = json.Unmarshal(out, &prData)
		if err != nil {
			logging.Warnf(
				"Unmarshal JSON from log for PR %d failed: %v",
				pr,
				err,
			)
			continue
		}
		authored, err := time.Parse(
			"Mon, 2 Jan 2006 15:04:05 -0700",
			prData.AuthoredString,
		)
		if err == nil {
			if authored.After(time.Now().Add(-90 * 24 * time.Hour)) {
				cmd = exec.Command("git", "branch", "--contains", hash)
				cmd.Dir = sourcesDir()
				out, err = cmd.CombinedOutput()
				if err != nil {
					logging.Warnf(
						"git branch contains for PR %d failed: %v",
						pr,
						err,
					)
					return err
				}
				branches := strings.Split(string(out), "\n")
				inMaster := false
				for _, branch := range branches {
					if strings.HasSuffix(
						strings.TrimRight(branch, " "),
						"master",
					) {
						inMaster = true
						break
					}
				}
				if inMaster {
					continue
				}
				// Yes we want this one!
				newGitLog = append(newGitLog, GitLogRecord{
					Authored:   authored,
					Committed:  authored,
					Subject:    fmt.Sprintf("PR #%d - %s", pr, prData.Subject),
					CommitHash: hash,
				})
			}
		} else {
			logging.Warnf("Authored date for PR %d could not be parsed: %v", pr, err)
		}
	}

	s.gitLog = newGitLog
	sort.Slice(s.gitLog, func(i, j int) bool {
		return s.gitLog[j].Authored.Before(s.gitLog[i].Authored)
	})

	// Shift PRs to the top
	prCount := 0
	for i := len(s.gitLog) - 1; i >= prCount; i-- {
		if strings.HasPrefix(s.gitLog[i].Subject, "PR #") {
			found := s.gitLog[i]
			s.gitLog = append(
				[]GitLogRecord{found},
				append(s.gitLog[:i], s.gitLog[i+1:]...)...)
			i++
			prCount++
		}
	}

	// Now show the first three non-PR commits on top and we're done
	prCommits := make([]GitLogRecord, prCount)
	nonPRCommits := make([]GitLogRecord, len(s.gitLog)-prCount)
	copy(prCommits, s.gitLog[:prCount])
	copy(nonPRCommits, s.gitLog[prCount:])
	s.gitLog = append(
		nonPRCommits[0:3],
		append(prCommits, nonPRCommits[3:]...)...)

	return nil
}

func (s *SourcesManager) cloneSources() error {
	s.sourcesLock.Lock()
	defer s.sourcesLock.Unlock()
	cmd := exec.Command(
		"git",
		"clone",
		"git@github.com:mit-dci/cbdc-universe0",
		sourcesDirName(),
	)
	cmd.Dir = sourcesParentDir()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(
			"Failed to clone sources. Do you have the right ssh keys configured? %v",
			err,
		)
	}

	cmd = exec.Command("git", "submodule", "update", "--init", "--recursive")
	cmd.Dir = sourcesDir()
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (s *SourcesManager) updateSources() error {
	s.sourcesLock.Lock()
	defer s.sourcesLock.Unlock()
	cmd := exec.Command("git", "checkout", "master")
	cmd.Dir = sourcesDir()
	err := cmd.Run()
	if err != nil {
		return err
	}
	cmd = exec.Command("git", "pull")
	cmd.Dir = sourcesDir()
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (s *SourcesManager) GetGitLog(offset, limit int) ([]GitLogRecord, error) {
	if offset >= len(s.gitLog) {
		return []GitLogRecord{}, ErrGitLogOutOfBounds
	}
	end := offset + limit
	if end >= len(s.gitLog) {
		end = len(s.gitLog) - 1
	}
	return s.gitLog[offset:end], nil
}

func (s *SourcesManager) CommitExists(hash string) bool {
	for _, c := range s.gitLog {
		if c.CommitHash == hash {
			return true
		}
	}
	return false
}
func (s *SourcesManager) ReadCommitArchive(hash string) ([]byte, error) {
	path, err := archivePath(hash)
	if err != nil {
		return nil, err
	}
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"source archive does not exist. Call MakeCommitArchive first!",
		)
	}
	return ioutil.ReadFile(path)
}

func (s *SourcesManager) CompileIfNeeded(
	hash string,
	profilingOrDebugging bool,
	progress chan float64,
) error {
	path, err := BinariesArchivePath(hash, profilingOrDebugging)
	if err != nil {
		if progress != nil {
			progress <- 100
			close(progress)
		}
		return err
	}
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return s.Compile(hash, profilingOrDebugging, progress)
	}
	if progress != nil {
		progress <- 100
		close(progress)
	}
	return nil
}

func (s *SourcesManager) MakeCommitArchive(hash string) error {
	s.sourcesLock.Lock()
	defer s.sourcesLock.Unlock()
	path, err := archivePath(hash)
	if err != nil {
		return err
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		// Already exists
		return nil
	}

	cmd := exec.Command("git", "checkout", hash)
	cmd.Dir = sourcesDir()
	err = cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("git", "submodule", "update", "--recursive")
	cmd.Dir = sourcesDir()
	err = cmd.Run()
	if err != nil {
		return err
	}

	return common.CreateArchive(sourcesDir(), path)
}
